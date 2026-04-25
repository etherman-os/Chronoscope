package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestExportUserData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully exports user data", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		userID := "user-123"
		sessionID := uuid.New().String()

		sessionRows := sqlmock.NewRows([]string{"id", "project_id", "user_id", "duration_ms", "video_path", "event_count", "error_count", "metadata", "status", "created_at", "completed_at"}).
			AddRow(sessionID, projectID, userID, 1000, nil, 2, 0, nil, "completed", time.Now(), nil)

		mock.ExpectQuery(`SELECT .* FROM sessions WHERE user_id = \$1 AND project_id = \$2`).
			WithArgs(userID, projectID).
			WillReturnRows(sessionRows)

		eventRows := sqlmock.NewRows([]string{"id", "session_id", "event_type", "timestamp_ms", "x", "y", "target", "payload", "created_at"}).
			AddRow(1, sessionID, "click", 1000, 10, 20, "btn", nil, time.Now())

		mock.ExpectQuery(`SELECT .* FROM events WHERE session_id = \$1 ORDER BY timestamp_ms ASC`).
			WithArgs(sessionID).
			WillReturnRows(eventRows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "user_id", Value: userID}}
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/export/"+userID, nil)

		ExportUserData(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["user_id"] != userID {
			t.Errorf("expected user_id %s, got %v", userID, resp["user_id"])
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		userID := "user-123"

		mock.ExpectQuery(`SELECT .* FROM sessions WHERE user_id = \$1 AND project_id = \$2`).
			WithArgs(userID, projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "user_id", Value: userID}}
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/export/"+userID, nil)

		ExportUserData(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})
}

func TestDeleteUserData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully deletes user data", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()
		cfg.Minio = mockMinIOTransport(t)

		projectID := uuid.New().String()
		userID := "user-123"
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT id FROM sessions WHERE user_id = \$1 AND project_id = \$2`).
			WithArgs(userID, projectID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(sessionID))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "user_id", Value: userID}}
		c.Request, _ = http.NewRequest("DELETE", "/v1/gdpr/delete/"+userID, nil)

		DeleteUserData(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		userID := "user-123"

		mock.ExpectQuery(`SELECT id FROM sessions WHERE user_id = \$1 AND project_id = \$2`).
			WithArgs(userID, projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "user_id", Value: userID}}
		c.Request, _ = http.NewRequest("DELETE", "/v1/gdpr/delete/"+userID, nil)

		DeleteUserData(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})
}

func TestListAuditLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully lists audit logs", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"id", "project_id", "action", "actor", "details", "created_at"}).
			AddRow(1, projectID, "session_initiated", "user-1", nil, time.Now())

		mock.ExpectQuery(`SELECT .* FROM audit_logs WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 20, 0).
			WillReturnRows(rows)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/audit-logs", nil)

		ListAuditLogs(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		logs, ok := resp["logs"].([]interface{})
		if !ok || len(logs) != 1 {
			t.Errorf("expected 1 log, got %v", resp["logs"])
		}
	})

	t.Run("database error on logs query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()

		mock.ExpectQuery(`SELECT .* FROM audit_logs WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 20, 0).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/audit-logs", nil)

		ListAuditLogs(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"id", "project_id", "action", "actor", "details", "created_at"}).
			AddRow(1, projectID, "action", "actor", nil, time.Now())

		mock.ExpectQuery(`SELECT .* FROM audit_logs WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 5, 10).
			WillReturnRows(rows)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/audit-logs?limit=5&offset=10", nil)

		ListAuditLogs(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("count error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"id", "project_id", "action", "actor", "details", "created_at"}).
			AddRow(1, projectID, "action", "actor", nil, time.Now())

		mock.ExpectQuery(`SELECT .* FROM audit_logs WHERE project_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(projectID, 20, 0).
			WillReturnRows(rows)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/gdpr/audit-logs", nil)

		ListAuditLogs(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})
}
