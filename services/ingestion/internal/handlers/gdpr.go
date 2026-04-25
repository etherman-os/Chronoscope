package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

const maxExportPageSize = 100

func ExportUserData(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		projectID, _ := c.Get("project_id")

		pageSize := defaultLimit
		if ps := c.Query("page_size"); ps != "" {
			if n, err := strconv.Atoi(ps); err == nil && n > 0 {
				pageSize = n
			}
		}
		if pageSize > maxExportPageSize {
			pageSize = maxExportPageSize
		}

		cursor := c.Query("cursor")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		var rows *sql.Rows
		var err error
		if cursor != "" {
			rows, err = cfg.DB.QueryContext(ctx,
				`SELECT id, project_id, user_id, duration_ms, video_path, event_count, error_count, metadata, status, created_at, completed_at FROM sessions WHERE user_id = $1 AND project_id = $2 AND id > $3 ORDER BY id ASC LIMIT $4`,
				userID, projectID, cursor, pageSize+1,
			)
		} else {
			rows, err = cfg.DB.QueryContext(ctx,
				`SELECT id, project_id, user_id, duration_ms, video_path, event_count, error_count, metadata, status, created_at, completed_at FROM sessions WHERE user_id = $1 AND project_id = $2 ORDER BY id ASC LIMIT $3`,
				userID, projectID, pageSize+1,
			)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sessions"})
			return
		}
		defer rows.Close()

		var sessions []map[string]interface{}
		totalEvents := 0

		for rows.Next() {
			var s models.Session
			if err := rows.Scan(&s.ID, &s.ProjectID, &s.UserID, &s.DurationMs, &s.VideoPath, &s.EventCount, &s.ErrorCount, &s.Metadata, &s.Status, &s.CreatedAt, &s.CompletedAt); err != nil {
				log.Printf("scan session row: %v", err)
				continue
			}

			eventRows, err := cfg.DB.QueryContext(ctx,
				`SELECT id, session_id, event_type, timestamp_ms, x, y, target, payload, created_at FROM events WHERE session_id = $1 ORDER BY timestamp_ms ASC LIMIT 10000`,
				s.ID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query events"})
				return
			}

			var events []map[string]interface{}
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
			if err := eventRows.Err(); err != nil {
				eventRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read events"})
				return
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
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read sessions"})
			return
		}

		hasMore := len(sessions) > pageSize
		if hasMore {
			sessions = sessions[:pageSize]
		}

		response := gin.H{
			"user_id":      userID,
			"sessions":     sessions,
			"total_events": totalEvents,
			"page_size":    pageSize,
		}
		if hasMore {
			response["next_cursor"] = sessions[len(sessions)-1]["id"].(string)
			response["has_more"] = true
		}
		c.JSON(http.StatusOK, response)
	}
}

func DeleteUserData(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		projectID, _ := c.Get("project_id")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		rows, err := cfg.DB.QueryContext(ctx,
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
				log.Printf("scan session id: %v", err)
				continue
			}
			sessionIDs = append(sessionIDs, sid)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read session ids"})
			return
		}

		deletedSessions := 0
		deletedEvents := 0

		for _, sid := range sessionIDs {
			var objectsToDelete []string
			opts := minio.ListObjectsOptions{
				Prefix:    sid + "/",
				Recursive: true,
			}
			for obj := range cfg.Minio.ListObjects(ctx, cfg.BucketName, opts) {
				if obj.Err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list objects in storage"})
					return
				}
				objectsToDelete = append(objectsToDelete, obj.Key)
			}

			var processedObjectsToDelete []string
			for obj := range cfg.Minio.ListObjects(ctx, cfg.ProcessedBucketName, opts) {
				if obj.Err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list processed objects in storage"})
					return
				}
				processedObjectsToDelete = append(processedObjectsToDelete, obj.Key)
			}

			tx, err := cfg.DB.BeginTx(ctx, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
				return
			}

			res, err := tx.ExecContext(ctx, `DELETE FROM events WHERE session_id = $1`, sid)
			if err != nil {
				_ = tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete events"})
				return
			}
			evCount, err := res.RowsAffected()
			if err != nil {
				_ = tx.Rollback()
				log.Printf("rows affected error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get deletion count"})
				return
			}

			_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sid)
			if err != nil {
				_ = tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
				return
			}

			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit deletion"})
				return
			}

			for _, objKey := range objectsToDelete {
				if err := cfg.Minio.RemoveObject(ctx, cfg.BucketName, objKey, minio.RemoveObjectOptions{}); err != nil {
					log.Printf("failed to delete object %s from storage: %v", objKey, err)
				}
			}
			for _, objKey := range processedObjectsToDelete {
				if err := cfg.Minio.RemoveObject(ctx, cfg.ProcessedBucketName, objKey, minio.RemoveObjectOptions{}); err != nil {
					log.Printf("failed to delete processed object %s from storage: %v", objKey, err)
				}
			}

			deletedSessions++
			deletedEvents += int(evCount)
		}

		if pid, ok := projectID.(string); ok {
			if err := LogAudit(ctx, cfg, pid, "gdpr_delete", "", map[string]interface{}{
				"user_id":          userID,
				"deleted_sessions": deletedSessions,
				"deleted_events":   deletedEvents,
			}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
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

		limit := defaultLimit
		offset := 0
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > maxLimit {
			limit = maxLimit
		}
		if o := c.Query("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := cfg.DB.QueryContext(ctx,
			`SELECT id, project_id, action, actor, details, created_at FROM audit_logs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			projectID, limit, offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit logs"})
			return
		}
		defer rows.Close()

		var logs []map[string]interface{}
		for rows.Next() {
			var id int64
			var pid string
			var action string
			var actor sql.NullString
			var details sql.NullString
			var createdAt time.Time

			if err := rows.Scan(&id, &pid, &action, &actor, &details, &createdAt); err != nil {
				log.Printf("scan audit log row: %v", err)
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
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audit logs"})
			return
		}

		var total int
		err = cfg.DB.QueryRowContext(ctx,
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
