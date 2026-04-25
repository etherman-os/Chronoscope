package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFunnel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid project returns 200 with funnel", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		mock.ExpectQuery(`SELECT COUNT\(DISTINCT s\.id\) FROM sessions s JOIN events e ON s\.id = e\.session_id WHERE s\.project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(80))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1 AND video_path IS NOT NULL AND video_path != ''`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(60))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1 AND status = 'completed'`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(40))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		funnel, ok := resp["funnel"].([]interface{})
		require.True(t, ok)
		require.Len(t, funnel, 4)
	})

	t.Run("missing project_id returns 401", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("database error on first query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("database error on second query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		mock.ExpectQuery(`SELECT COUNT\(DISTINCT s\.id\) FROM sessions s JOIN events e ON s\.id = e\.session_id WHERE s\.project_id = \$1`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("database error on third query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		mock.ExpectQuery(`SELECT COUNT\(DISTINCT s\.id\) FROM sessions s JOIN events e ON s\.id = e\.session_id WHERE s\.project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(80))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1 AND video_path IS NOT NULL AND video_path != ''`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("database error on fourth query returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
		mock.ExpectQuery(`SELECT COUNT\(DISTINCT s\.id\) FROM sessions s JOIN events e ON s\.id = e\.session_id WHERE s\.project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(80))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1 AND video_path IS NOT NULL AND video_path != ''`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(60))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM sessions WHERE project_id = \$1 AND status = 'completed'`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/funnel", nil)

		GetFunnel(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
