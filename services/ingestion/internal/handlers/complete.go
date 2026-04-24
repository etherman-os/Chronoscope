package handlers

import (
	"log"
	"net/http"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
)

// CompleteSession marks a session as completed.
func CompleteSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		authenticatedProjectID, _ := c.Get("project_id")
		authPID, _ := authenticatedProjectID.(string)
		var ownerProjectID string
		err := cfg.DB.QueryRow(`SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&ownerProjectID)
		if err != nil || ownerProjectID != authPID {
			c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to project"})
			return
		}

		_, err = cfg.DB.Exec(
			`UPDATE sessions SET status = 'completed', completed_at = NOW() WHERE id = $1`,
			sessionID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete session"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRow(`SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			if err := LogAudit(cfg, projectID, "session_completed", "", map[string]interface{}{"session_id": sessionID}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	}
}
