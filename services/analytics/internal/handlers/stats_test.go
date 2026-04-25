package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chronoscope/analytics/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) (*config.Config, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	cfg := &config.Config{
		ServerAddr: ":8081",
		DB:         db,
	}
	return cfg, mock
}

func TestGetSessionStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid project returns 200 with stats", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COALESCE\(AVG\(duration_ms\), 0\), COUNT\(\*\), COALESCE\(SUM\(event_count\), 0\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"avg", "count", "sum"}).AddRow(1234.5, 10, 100))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/sessions/stats", nil)

		GetSessionStats(cfg)(c)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		stats, ok := resp["stats"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 1234.5, stats["avg_duration_ms"])
		assert.Equal(t, float64(10), stats["total_sessions"])
		assert.Equal(t, float64(100), stats["total_events"])
		assert.Equal(t, float64(10), stats["avg_events_per_session"])
	})

	t.Run("missing project_id returns 401", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)
		defer cfg.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/sessions/stats", nil)

		GetSessionStats(cfg)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("database error returns 500", func(t *testing.T) {
		cfg, mock := setupTestConfig(t)
		defer cfg.DB.Close()

		projectID := uuid.New().String()
		mock.ExpectQuery(`SELECT COALESCE\(AVG\(duration_ms\), 0\), COUNT\(\*\), COALESCE\(SUM\(event_count\), 0\) FROM sessions WHERE project_id = \$1`).
			WithArgs(projectID).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("project_id", projectID)
		c.Request, _ = http.NewRequest("GET", "/v1/analytics/sessions/stats", nil)

		GetSessionStats(cfg)(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
