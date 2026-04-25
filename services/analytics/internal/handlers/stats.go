package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/chronoscope/analytics/internal/config"
	"github.com/gin-gonic/gin"
)

// SessionStats holds aggregate session statistics.
type SessionStats struct {
	AvgDurationMs       float64 `json:"avg_duration_ms"`
	TotalSessions       int     `json:"total_sessions"`
	TotalEvents         int     `json:"total_events"`
	AvgEventsPerSession float64 `json:"avg_events_per_session"`
}

// GetSessionStats returns aggregate session stats for a project.
func GetSessionStats(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, exists := c.Get("project_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var stats SessionStats
		err := cfg.DB.QueryRowContext(ctx, `
			SELECT
				COALESCE(AVG(duration_ms), 0),
				COUNT(*),
				COALESCE(SUM(event_count), 0)
			FROM sessions
			WHERE project_id = $1
		`, projectID).Scan(
			&stats.AvgDurationMs,
			&stats.TotalSessions,
			&stats.TotalEvents,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query session stats"})
			return
		}

		if stats.TotalSessions > 0 {
			stats.AvgEventsPerSession = float64(stats.TotalEvents) / float64(stats.TotalSessions)
		}

		c.JSON(http.StatusOK, gin.H{
			"project_id": projectID,
			"stats":      stats,
		})
	}
}
