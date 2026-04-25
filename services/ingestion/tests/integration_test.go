package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/handlers"
	"github.com/chronoscope/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func setupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/v1")
	v1.Use(middleware.APIKeyAuth(cfg.DB))

	v1.POST("/sessions/init", handlers.InitSession(cfg))
	v1.POST("/sessions/:id/chunks", handlers.UploadChunk(cfg))
	v1.POST("/sessions/:id/events", handlers.UploadEvents(cfg))
	v1.POST("/sessions/:id/complete", handlers.CompleteSession(cfg))
	v1.GET("/sessions", handlers.ListSessions(cfg))
	v1.GET("/sessions/:id", handlers.GetSession(cfg))

	return router
}

func mockMinIOClient(t *testing.T) *minio.Client {
	t.Helper()
	client, err := minio.New("localhost:1", &minio.Options{
		Creds:  credentials.NewStaticV4("test", "test", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("failed to create minio client: %v", err)
	}
	return client
}

func expectAuthLookup(mock sqlmock.Sqlmock, projectID string) {
	mock.ExpectQuery(`SELECT id, api_key_hash FROM projects WHERE api_key_hash IS NOT NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "api_key_hash"}).
			AddRow(projectID, "$2a$10$gf.NT4vS/mClWzg0r1mAweLDwDNX1v5faKle.NamtfrmxTOveZ.O2"))
}

func TestFullSessionLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		ServerAddr: ":8080",
		DB:         db,
		Minio:      mockMinIOClient(t),
		BucketName: "chronoscope-test",
	}

	apiKey := "valid-api-key-123"
	projectID := "22222222-2222-2222-2222-222222222222"

	expectAuthLookup(mock, projectID)

	mock.ExpectExec(`INSERT INTO sessions`).
		WithArgs(sqlmock.AnyArg(), projectID, "integration-test", "capturing", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(projectID, "session_initiated", "integration-test", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	router := setupRouter(cfg)

	initBody, _ := json.Marshal(map[string]interface{}{
		"user_id":      "integration-test",
		"capture_mode": "video",
	})
	req := httptest.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(initBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("init session failed: status=%d body=%s", w.Code, w.Body.String())
	}
	var initResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &initResp); err != nil {
		t.Fatalf("failed to parse init response: %v", err)
	}
	createdSessionID, ok := initResp["session_id"].(string)
	if !ok || createdSessionID == "" {
		t.Fatal("expected session_id in init response")
	}

	expectAuthLookup(mock, projectID)

	mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
		WithArgs(createdSessionID).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO events`)
	mock.ExpectExec(`INSERT INTO events`).
		WithArgs(createdSessionID, "click", 1000, 100, 200, "", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE sessions`).
		WithArgs(1, createdSessionID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
		WithArgs(createdSessionID).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(projectID, "events_uploaded", "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	eventsBody, _ := json.Marshal(map[string]interface{}{
		"events": []map[string]interface{}{
			{"event_type": "click", "timestamp_ms": 1000, "x": 100, "y": 200},
		},
	})
	req = httptest.NewRequest("POST", "/v1/sessions/"+createdSessionID+"/events", bytes.NewReader(eventsBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload events failed: status=%d body=%s", w.Code, w.Body.String())
	}

	expectAuthLookup(mock, projectID)

	mock.ExpectQuery(`SELECT project_id, status FROM sessions WHERE id = \$1`).
		WithArgs(createdSessionID).
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "status"}).AddRow(projectID, "capturing"))
	mock.ExpectExec(`UPDATE sessions`).
		WithArgs(createdSessionID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
		WithArgs(createdSessionID).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(projectID, "session_completed", "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req = httptest.NewRequest("POST", "/v1/sessions/"+createdSessionID+"/complete", nil)
	req.Header.Set("X-API-Key", apiKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("complete session failed: status=%d body=%s", w.Code, w.Body.String())
	}

	expectAuthLookup(mock, projectID)

	rows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
		AddRow(createdSessionID, projectID, "integration-test", nil, nil, 1, 0, nil, "completed", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT .* FROM sessions WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(projectID, 20, 0).
		WillReturnRows(rows)

	req = httptest.NewRequest("GET", "/v1/sessions?project_id="+projectID, nil)
	req.Header.Set("X-API-Key", apiKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list sessions failed: status=%d body=%s", w.Code, w.Body.String())
	}
	var listResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	sessions, ok := listResp["sessions"].([]interface{})
	if !ok || len(sessions) == 0 {
		t.Fatal("expected at least one session in list")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestChunkUploadValidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		ServerAddr: ":8080",
		DB:         db,
		Minio:      mockMinIOClient(t),
		BucketName: "chronoscope-test",
	}

	apiKey := "valid-api-key-123"
	projectID := "22222222-2222-2222-2222-222222222222"
	sessionID := uuid.New().String()

	expectAuthLookup(mock, projectID)
	mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
		WithArgs(sessionID).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

	router := setupRouter(cfg)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, _ := writer.CreateFormFile("chunk", "chunk.jpg")
	_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46})
	writer.Close()

	req := httptest.NewRequest("POST", "/v1/sessions/"+sessionID+"/chunks", &b)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Chunk-Index", "0")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusForbidden {
		t.Fatalf("unexpected validation failure: status=%d body=%s", w.Code, w.Body.String())
	}
}
