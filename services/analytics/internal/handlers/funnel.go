package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/chronoscope/analytics/internal/config"
	"github.com/gin-gonic/gin"
)

// FunnelStage represents a single stage in the session completion funnel.
type FunnelStage struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

// GetFunnel returns the session completion funnel for a project.
func GetFunnel(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, exists := c.Get("project_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var totalSessions int
		err := cfg.DB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sessions WHERE project_id = $1",
			projectID,
		).Scan(&totalSessions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query total sessions"})
			return
		}

		var sessionsWithEvents int
		err = cfg.DB.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT s.id)
			FROM sessions s
			JOIN events e ON s.id = e.session_id
			WHERE s.project_id = $1
		`, projectID).Scan(&sessionsWithEvents)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sessions with events"})
			return
		}

		var sessionsWithChunks int
		err = cfg.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sessions
			WHERE project_id = $1 AND video_path IS NOT NULL AND video_path != ''
		`, projectID).Scan(&sessionsWithChunks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sessions with chunks"})
			return
		}

		var completedSessions int
		err = cfg.DB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sessions WHERE project_id = $1 AND status = 'completed'",
			projectID,
		).Scan(&completedSessions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query completed sessions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"project_id": projectID,
			"funnel": []FunnelStage{
				{Stage: "total_sessions", Count: totalSessions},
				{Stage: "sessions_with_events", Count: sessionsWithEvents},
				{Stage: "sessions_with_chunks", Count: sessionsWithChunks},
				{Stage: "completed_sessions", Count: completedSessions},
			},
		})
	}
}
