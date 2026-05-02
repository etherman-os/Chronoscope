package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// GetVideo streams the processed session video through the authenticated API.
func GetVideo(cfg *config.Config) gin.HandlerFunc {
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

		objectName := sessionID + "/session.mp4"
		info, err := cfg.Minio.StatObject(ctx, cfg.ProcessedBucketName, objectName, minio.StatObjectOptions{})
		if err != nil {
			var minioErr minio.ErrorResponse
			if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
				c.JSON(http.StatusNotFound, gin.H{"error": "processed video not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stat video"})
			return
		}

		object, err := cfg.Minio.GetObject(ctx, cfg.ProcessedBucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read video"})
			return
		}
		defer object.Close()

		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.DataFromReader(http.StatusOK, info.Size, "video/mp4", object, nil)
	}
}
