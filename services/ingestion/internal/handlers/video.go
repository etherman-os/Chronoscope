package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
)

// GetVideo returns a presigned URL for the processed session video.
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

		// Presigned URL valid for 15 minutes
		presignedURL, err := cfg.Minio.PresignedGetObject(ctx, cfg.ProcessedBucketName, sessionID+"/session.mp4", 15*time.Minute, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate video URL"})
			return
		}

		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Redirect(http.StatusFound, presignedURL.String())
	}
}
