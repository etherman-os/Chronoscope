package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
)

const maxEventBatchSize = 1000

type uploadEventsRequest struct {
	Events []struct {
		EventType   string          `json:"event_type"`
		TimestampMs int             `json:"timestamp_ms"`
		X           int             `json:"x"`
		Y           int             `json:"y"`
		Target      string          `json:"target"`
		Payload     json.RawMessage `json:"payload"`
	} `json:"events"`
}

// UploadEvents persists a batch of events and increments the session event_count.
func UploadEvents(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		authenticatedProjectID, _ := c.Get("project_id")
		authPID, _ := authenticatedProjectID.(string)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var ownerProjectID string
		err := cfg.DB.QueryRowContext(ctx, `SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&ownerProjectID)
		if err != nil || ownerProjectID != authPID {
			c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to project"})
			return
		}

		if c.GetHeader("Content-Type") != "application/json" {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
			return
		}

		var req uploadEventsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.Events) == 0 {
			c.JSON(http.StatusOK, gin.H{"count": 0})
			return
		}

		if len(req.Events) > maxEventBatchSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "event batch exceeds maximum size"})
			return
		}

		tx, err := cfg.DB.BeginTx(ctx, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO events (session_id, event_type, timestamp_ms, x, y, target, payload) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare statement"})
			return
		}
		defer stmt.Close()

		for _, ev := range req.Events {
			var payload interface{}
			if len(ev.Payload) > 0 {
				payload = []byte(ev.Payload)
			} else {
				payload = nil
			}

			_, err := stmt.ExecContext(ctx, sessionID, ev.EventType, ev.TimestampMs, ev.X, ev.Y, ev.Target, payload)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert event"})
				return
			}
		}

		_, err = tx.ExecContext(ctx,
			`UPDATE sessions SET event_count = event_count + $1 WHERE id = $2`,
			len(req.Events),
			sessionID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session event count"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRowContext(ctx, `SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			if err := LogAudit(cfg, projectID, "events_uploaded", "", map[string]interface{}{"session_id": sessionID, "event_count": len(req.Events)}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{"count": len(req.Events)})
	}
}
