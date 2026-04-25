package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultLimit = 20
)

type initSessionRequest struct {
	UserID      string                 `json:"user_id" binding:"required"`
	CaptureMode string                 `json:"capture_mode" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type initSessionResponse struct {
	SessionID string    `json:"session_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InitSession creates a new capture session.
func InitSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Content-Type") != "application/json" {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
			return
		}

		var req initSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		projectID, _ := c.Get("project_id")
		sessionID := uuid.New().String()

		// Merge capture_mode into metadata for storage.
		meta := req.Metadata
		if meta == nil {
			meta = make(map[string]interface{})
		}
		meta["capture_mode"] = req.CaptureMode

		metadataJSON, err := json.Marshal(meta)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal metadata"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		_, err = cfg.DB.ExecContext(ctx,
			`INSERT INTO sessions (id, project_id, user_id, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			sessionID,
			projectID,
			req.UserID,
			"capturing",
			metadataJSON,
			time.Now(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}

		if pid, ok := projectID.(string); ok {
			if err := LogAudit(cfg, pid, "session_initiated", req.UserID, map[string]interface{}{"session_id": sessionID}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		expiresAt := time.Now().Add(1 * time.Hour)

		c.JSON(http.StatusCreated, initSessionResponse{
			SessionID: sessionID,
			UploadURL: "/v1/sessions/" + sessionID + "/chunks",
			ExpiresAt: expiresAt,
		})
	}
}

// ListSessions returns paginated sessions for a project.
func ListSessions(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, ok := c.Get("project_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
			return
		}
		pid, ok := projectID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid project context"})
			return
		}

		limit := defaultLimit
		offset := 0
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		if o := c.Query("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := cfg.DB.QueryContext(ctx,
			`SELECT id, project_id, user_id, duration_ms, video_path, event_count, error_count, metadata, status, created_at, completed_at FROM sessions WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			pid, limit, offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
			return
		}
		defer rows.Close()

		var sessions []models.Session
		for rows.Next() {
			var s models.Session
			err := rows.Scan(&s.ID, &s.ProjectID, &s.UserID, &s.DurationMs, &s.VideoPath, &s.EventCount, &s.ErrorCount, &s.Metadata, &s.Status, &s.CreatedAt, &s.CompletedAt)
			if err != nil {
				log.Printf("scan session row: %v", err)
				continue
			}
			sessions = append(sessions, s)
		}

		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

// GetSession retrieves a single session and its associated events.
func GetSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		authenticatedProjectID, _ := c.Get("project_id")
		authPID, _ := authenticatedProjectID.(string)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var s models.Session
		err := cfg.DB.QueryRowContext(ctx,
			`SELECT id, project_id, user_id, duration_ms, video_path, event_count, error_count, metadata, status, created_at, completed_at FROM sessions WHERE id = $1`,
			sessionID,
		).Scan(&s.ID, &s.ProjectID, &s.UserID, &s.DurationMs, &s.VideoPath, &s.EventCount, &s.ErrorCount, &s.Metadata, &s.Status, &s.CreatedAt, &s.CompletedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
			return
		}

		if s.ProjectID != authPID {
			c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to project"})
			return
		}

		rows, err := cfg.DB.QueryContext(ctx,
			`SELECT id, session_id, event_type, timestamp_ms, x, y, target, payload, created_at FROM events WHERE session_id = $1 ORDER BY timestamp_ms ASC`,
			sessionID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get events"})
			return
		}
		defer rows.Close()

		var events []map[string]interface{}
		for rows.Next() {
			var eventID int64
			var sid string
			var eventType string
			var timestampMs sql.NullInt64
			var x, y sql.NullInt32
			var target sql.NullString
			var payload sql.NullString
			var createdAt time.Time

			err := rows.Scan(&eventID, &sid, &eventType, &timestampMs, &x, &y, &target, &payload, &createdAt)
			if err != nil {
				log.Printf("scan event row: %v", err)
				continue
			}

			ev := map[string]interface{}{
				"id":           eventID,
				"session_id":   sid,
				"event_type":   eventType,
				"timestamp_ms": timestampMs,
				"x":            x,
				"y":            y,
				"target":       target,
				"payload":      payload,
				"created_at":   createdAt,
			}
			events = append(events, ev)
		}

		c.JSON(http.StatusOK, gin.H{
			"session": s,
			"events":  events,
		})
	}
}
