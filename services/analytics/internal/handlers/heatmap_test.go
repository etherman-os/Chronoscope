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

func TestGetHeatmap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid project returns 200 with points", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"x", "y", "count"}).
			AddRow(100, 200, 5).
			AddRow(50, 75, 3)

		mock.ExpectQuery(`SELECT e\.x, e\.y, COUNT\(\*\) as count FROM events e JOIN sessions s ON e\.session_id = s\.id WHERE s\.project_id = \$1 GROUP BY e\.x, e\.y ORDER BY count DESC LIMIT 100`).
			WithArgs(projectID).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/heatmap", nil)

		GetHeatmap(cfg)(c)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		points, ok := resp["points"].([]interface{})
		require.True(t, ok)
		require.Len(t, points, 2)
	})

	t.Run("missing project_id returns 401", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/heatmap", nil)

		GetHeatmap(cfg)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT e\.x, e\.y, COUNT\(\*\) as count FROM events e JOIN sessions s ON e\.session_id = s\.id WHERE s\.project_id = \$1 GROUP BY e\.x, e\.y ORDER BY count DESC LIMIT 100`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/heatmap", nil)

		GetHeatmap(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("scan error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		rows := sqlmock.NewRows([]string{"x", "y", "count"}).
			AddRow("not-an-int", 200, 5)

		mock.ExpectQuery(`SELECT e\.x, e\.y, COUNT\(\*\) as count FROM events e JOIN sessions s ON e\.session_id = s\.id WHERE s\.project_id = \$1 GROUP BY e\.x, e\.y ORDER BY count DESC LIMIT 100`).
			WithArgs(projectID).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/heatmap", nil)

		GetHeatmap(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
