package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type completeSessionRequest struct {
	DurationMs *int `json:"duration_ms"`
}

// CompleteSession marks a session as completed.
func CompleteSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		authenticatedProjectID, _ := c.Get("project_id")
		authPID, _ := authenticatedProjectID.(string)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var ownerProjectID string
		var status string
		err := cfg.DB.QueryRowContext(ctx, `SELECT project_id, status FROM sessions WHERE id = $1`, sessionID).Scan(&ownerProjectID, &status)
		if err != nil || ownerProjectID != authPID {
			c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to project"})
			return
		}

		var req completeSessionRequest
		if c.Request.ContentLength > 0 {
			contentType := c.GetHeader("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
				return
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		if status == "completed" || status == "processing" || status == "ready" {
			c.JSON(http.StatusOK, gin.H{"status": status})
			return
		}

		if req.DurationMs != nil && *req.DurationMs >= 0 {
			_, err = cfg.DB.ExecContext(ctx,
				`UPDATE sessions SET status = 'completed', duration_ms = $2, completed_at = NOW() WHERE id = $1`,
				sessionID,
				*req.DurationMs,
			)
		} else {
			_, err = cfg.DB.ExecContext(ctx,
				`UPDATE sessions SET status = 'completed', completed_at = NOW() WHERE id = $1`,
				sessionID,
			)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete session"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRowContext(ctx, `SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			if err := LogAudit(ctx, cfg, projectID, "session_completed", "", map[string]interface{}{"session_id": sessionID}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		if cfg.Redis != nil {
			if err := cfg.Redis.XAdd(ctx, &redis.XAddArgs{
				Stream: "chronoscope:process_queue",
				Values: map[string]interface{}{"session_id": sessionID},
			}).Err(); err != nil {
				log.Printf("failed to enqueue session for processing: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue session"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	}
}
