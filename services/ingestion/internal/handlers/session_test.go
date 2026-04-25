package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupTestConfig(t *testing.T) (*config.Config, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	cfg := &config.Config{
		ServerAddr: ":8080",
		DB:         db,
		Minio:      nil,
		BucketName: "chronoscope-test",
	}
	return cfg, mock
}

func TestInitSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request returns 201", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectExec(`INSERT INTO sessions`).
			WithArgs(sqlmock.AnyArg(), projectID, "user-123", "capturing", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(projectID, "session_initiated", "user-123", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		body, _ := json.Marshal(map[string]interface{}{
			"user_id":      "user-123",
			"capture_mode": "video",
			"metadata":     map[string]interface{}{"browser": "chrome"},
		})
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		InitSession(cfg)(c)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["session_id"] == "" {
			t.Error("expected session_id in response")
		}
		if resp["upload_url"] == "" {
			t.Error("expected upload_url in response")
		}
	})

	t.Run("missing API key context still processes if project_id is set", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectExec(`INSERT INTO sessions`).
			WithArgs(sqlmock.AnyArg(), projectID, "user-456", "capturing", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(projectID, "session_initiated", "user-456", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		body, _ := json.Marshal(map[string]interface{}{
			"user_id":      "user-456",
			"capture_mode": "video",
		})
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		InitSession(cfg)(c)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", uuid.New().String())
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", strings.NewReader("not json"))
		c.Request.Header.Set("Content-Type", "application/json")

		InitSession(cfg)(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", uuid.New().String())
		body, _ := json.Marshal(map[string]interface{}{"user_id": "user-123"})
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		InitSession(cfg)(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestListSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid project_id returns 200", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(uuid.New().String(), projectID, "user-1", nil, nil, 10, 0, nil, "completed", time.Now(), nil)

		mock.ExpectQuery(`SELECT .* FROM sessions WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 20, 0).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions?project_id="+projectID, nil)

		ListSessions(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		sessions, ok := resp["sessions"].([]interface{})
		if !ok || len(sessions) != 1 {
			t.Errorf("expected 1 session, got %v", resp["sessions"])
		}
	})

	t.Run("cross-project access blocked", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		authProjectID := uuid.New().String()
		otherProjectID := uuid.New().String()
		sessionID := uuid.New().String()

		// GetSession query
		rows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(sessionID, otherProjectID, "user-1", nil, nil, 0, 0, nil, "capturing", time.Now(), nil)
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", authProjectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("GET", "/v1/sessions/"+sessionID, nil)

		GetSession(cfg)(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 20, 0).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions?project_id="+projectID, nil)

		ListSessions(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(uuid.New().String(), projectID, "user-1", nil, nil, 10, 0, nil, "completed", time.Now(), nil)

		mock.ExpectQuery(`SELECT .* FROM sessions WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 5, 10).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions?limit=5&offset=10", nil)

		ListSessions(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})
}

func TestGetSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("session not found returns 404", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE id = \$1`).
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
		c.Request, _ = http.NewRequest("GET", "/v1/sessions/nonexistent", nil)

		GetSession(cfg)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("session found returns 200 with events", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		sessionRows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(sessionID, projectID, "user-1", nil, nil, 2, 0, nil, "capturing", time.Now(), nil)
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sessionRows)

		eventRows := sqlmock.NewRows([]string{"id", "session_id", "event_type", "timestamp_ms", "x", "y", "target", "payload", "created_at"}).
			AddRow(1, sessionID, "click", 1000, 100, 200, "button", nil, time.Now()).
			AddRow(2, sessionID, "scroll", 2000, 0, 500, "window", nil, time.Now())
		mock.ExpectQuery(`SELECT .* FROM events WHERE session_id = \$1 ORDER BY timestamp_ms ASC`).
			WithArgs(sessionID).
			WillReturnRows(eventRows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("GET", "/v1/sessions/"+sessionID, nil)

		GetSession(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["session"] == nil {
			t.Error("expected session in response")
		}
		events, ok := resp["events"].([]interface{})
		if !ok || len(events) != 2 {
			t.Errorf("expected 2 events, got %v", resp["events"])
		}
	})

	t.Run("database error on session query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("GET", "/v1/sessions/"+sessionID, nil)

		GetSession(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("database error on events query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		sessionRows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(sessionID, projectID, "user-1", nil, nil, 2, 0, nil, "capturing", time.Now(), nil)
		mock.ExpectQuery(`SELECT .* FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sessionRows)
		mock.ExpectQuery(`SELECT .* FROM events WHERE session_id = \$1 ORDER BY timestamp_ms ASC`).
			WithArgs(sessionID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("GET", "/v1/sessions/"+sessionID, nil)

		GetSession(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

func TestInitSessionErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing content-type returns 415", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", uuid.New().String())
		body, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "capture_mode": "video"})
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(body))

		InitSession(cfg)(c)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status %d, got %d", http.StatusUnsupportedMediaType, w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectExec(`INSERT INTO sessions`).
			WithArgs(sqlmock.AnyArg(), projectID, "user-123", "capturing", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		body, _ := json.Marshal(map[string]interface{}{
			"user_id":      "user-123",
			"capture_mode": "video",
		})
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/init", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		InitSession(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}
