package handlers

import (
	"net/http"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
)

// CompleteSession marks a session as completed.
func CompleteSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		_, err := cfg.DB.Exec(
			`UPDATE sessions SET status = 'completed', completed_at = NOW() WHERE id = $1`,
			sessionID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete session"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRow(`SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			_ = LogAudit(cfg, projectID, "session_completed", "", map[string]interface{}{"session_id": sessionID})
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	}
}
