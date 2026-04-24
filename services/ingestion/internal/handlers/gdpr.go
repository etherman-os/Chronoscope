package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

func ExportUserData(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		projectID, _ := c.Get("project_id")

		rows, err := cfg.DB.Query(
			`SELECT id, project_id, user_id, duration_ms, video_path, event_count, error_count, metadata, status, created_at, completed_at FROM sessions WHERE user_id = $1 AND project_id = $2`,
			userID, projectID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sessions"})
			return
		}
		defer rows.Close()

		sessions := []map[string]interface{}{}
		totalEvents := 0

		for rows.Next() {
			var s models.Session
			if err := rows.Scan(&s.ID, &s.ProjectID, &s.UserID, &s.DurationMs, &s.VideoPath, &s.EventCount, &s.ErrorCount, &s.Metadata, &s.Status, &s.CreatedAt, &s.CompletedAt); err != nil {
				continue
			}

			eventRows, err := cfg.DB.Query(
				`SELECT id, session_id, event_type, timestamp_ms, x, y, target, payload, created_at FROM events WHERE session_id = $1 ORDER BY timestamp_ms ASC`,
				s.ID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query events"})
				return
			}

			events := []map[string]interface{}{}
			for eventRows.Next() {
				var eventID int64
				var sid string
				var eventType string
				var timestampMs sql.NullInt64
				var x, y sql.NullInt32
				var target sql.NullString
				var payload sql.NullString
				var createdAt time.Time

				if err := eventRows.Scan(&eventID, &sid, &eventType, &timestampMs, &x, &y, &target, &payload, &createdAt); err != nil {
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
			eventRows.Close()

			totalEvents += len(events)

			sessions = append(sessions, map[string]interface{}{
				"id":           s.ID,
				"project_id":   s.ProjectID,
				"user_id":      s.UserID,
				"duration_ms":  s.DurationMs,
				"video_path":   s.VideoPath,
				"event_count":  s.EventCount,
				"error_count":  s.ErrorCount,
				"metadata":     s.Metadata,
				"status":       s.Status,
				"created_at":   s.CreatedAt,
				"completed_at": s.CompletedAt,
				"events":       events,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":      userID,
			"sessions":     sessions,
			"total_events": totalEvents,
		})
	}
}

func DeleteUserData(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		projectID, _ := c.Get("project_id")

		rows, err := cfg.DB.Query(
			`SELECT id FROM sessions WHERE user_id = $1 AND project_id = $2`,
			userID, projectID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sessions"})
			return
		}
		defer rows.Close()

		var sessionIDs []string
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err != nil {
				continue
			}
			sessionIDs = append(sessionIDs, sid)
		}

		deletedSessions := 0
		deletedEvents := 0

		for _, sid := range sessionIDs {
			opts := minio.ListObjectsOptions{
				Prefix:    sid + "/",
				Recursive: true,
			}
			for obj := range cfg.Minio.ListObjects(context.Background(), cfg.BucketName, opts) {
				if obj.Err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list objects in storage"})
					return
				}
				if err := cfg.Minio.RemoveObject(context.Background(), cfg.BucketName, obj.Key, minio.RemoveObjectOptions{}); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete object from storage"})
					return
				}
			}

			tx, err := cfg.DB.Begin()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
				return
			}

			res, err := tx.Exec(`DELETE FROM events WHERE session_id = $1`, sid)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete events"})
				return
			}
			evCount, _ := res.RowsAffected()

			_, err = tx.Exec(`DELETE FROM sessions WHERE id = $1`, sid)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
				return
			}

			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit deletion"})
				return
			}

			deletedSessions++
			deletedEvents += int(evCount)
		}

		if pid, ok := projectID.(string); ok {
			_ = LogAudit(cfg, pid, "gdpr_delete", "", map[string]interface{}{
				"user_id":          userID,
				"deleted_sessions": deletedSessions,
				"deleted_events":   deletedEvents,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"deleted_sessions": deletedSessions,
			"deleted_events":   deletedEvents,
		})
	}
}

func ListAuditLogs(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, _ := c.Get("project_id")

		limit := 20
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

		rows, err := cfg.DB.Query(
			`SELECT id, project_id, action, actor, details, created_at FROM audit_logs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			projectID, limit, offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit logs"})
			return
		}
		defer rows.Close()

		logs := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var pid string
			var action string
			var actor sql.NullString
			var details sql.NullString
			var createdAt time.Time

			if err := rows.Scan(&id, &pid, &action, &actor, &details, &createdAt); err != nil {
				continue
			}

			logs = append(logs, map[string]interface{}{
				"id":         id,
				"project_id": pid,
				"action":     action,
				"actor":      actor,
				"details":    details,
				"created_at": createdAt,
			})
		}

		var total int
		err = cfg.DB.QueryRow(
			`SELECT COUNT(*) FROM audit_logs WHERE project_id = $1`,
			projectID,
		).Scan(&total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":  logs,
			"total": total,
		})
	}
}
