package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestCompleteSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully completes session", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
		mock.ExpectExec(`UPDATE sessions SET status = 'completed', completed_at = NOW\(\) WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
		mock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(projectID, "session_completed", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/complete", nil)

		CompleteSession(cfg)(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
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

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", authPID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/complete", nil)

		CompleteSession(cfg)(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
		}
	})

	t.Run("session not found returns 403", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnError(sql.ErrNoRows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/complete", nil)

		CompleteSession(cfg)(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
		}
	})

	t.Run("update failure returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		sessionID := uuid.New().String()

		mock.ExpectQuery(`SELECT project_id FROM sessions WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow(projectID))
		mock.ExpectExec(`UPDATE sessions SET status = 'completed', completed_at = NOW\(\) WHERE id = \$1`).
			WithArgs(sessionID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Params = gin.Params{{Key: "id", Value: sessionID}}
		c.Request, _ = http.NewRequest("POST", "/v1/sessions/"+sessionID+"/complete", nil)

		CompleteSession(cfg)(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, w.Code, w.Body.String())
		}
	})
}
