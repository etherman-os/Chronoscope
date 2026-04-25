package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestUploadEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully uploads events", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		body, _ := json.Marshal(map[string]interface{}{
			"events": []map[string]interface{}{
				{"event_type": "click", "timestamp_ms": 1000, "x": 10, "y": 20, "target": "btn"},
			},
		})

		mock.ExpectBegin()
		mock.ExpectPrepare(`INSERT INTO events`)
		mock.ExpectExec(`INSERT INTO events`).
			WithArgs(sessionID, "click", 1000, 10, 20, "btn", nil).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`UPDATE sessions SET event_count = event_count \+ \$1 WHERE id = \$2`).
			WithArgs(1, sessionID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
		mock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(projectID, "events_uploaded", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("missing content-type returns 415", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", strings.NewReader(""))

		UploadEvents(cfg)(c)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status %d, got %d: %s", http.StatusUnsupportedMediaType, w.Code, w.Body.String())
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", strings.NewReader("not json"))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("empty events returns 200 with count 0", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		body, _ := json.Marshal(map[string]interface{}{"events": []map[string]interface{}{}})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["count"] != float64(0) {
			t.Errorf("expected count 0, got %v", resp["count"])
		}
	})

	t.Run("event batch too large returns 413", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		events := make([]map[string]interface{}, maxEventBatchSize+1)
		for i := range events {
			events[i] = map[string]interface{}{"event_type": "click", "timestamp_ms": i}
		}
		body, _ := json.Marshal(map[string]interface{}{"events": events})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status %d, got %d: %s", http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
		}
	})

	t.Run("wrong project returns 403", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		authPID := uuid.New().String()
		ownerPID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(ownerPID))

		body, _ := json.Marshal(map[string]interface{}{
			"events": []map[string]interface{}{
				{"event_type": "click", "timestamp_ms": 1000},
			},
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", authPID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
		}
	})

	t.Run("transaction begin failure returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))

		body, _ := json.Marshal(map[string]interface{}{
			"events": []map[string]interface{}{
				{"event_type": "click", "timestamp_ms": 1000},
			},
		})

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/events", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		UploadEvents(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})
}
