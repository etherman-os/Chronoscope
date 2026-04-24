package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

const maxChunkSize = 2 << 20 // 2 MiB

// UploadChunk handles multipart chunk uploads to MinIO.
func UploadChunk(cfg *config.Config) gin.HandlerFunc {
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

		chunkIndexStr := c.GetHeader("X-Chunk-Index")
		if chunkIndexStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-Index header is required"})
			return
		}

		chunkIndex, err := strconv.Atoi(chunkIndexStr)
		if err != nil || chunkIndex < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-Chunk-Index"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChunkSize)
		file, header, err := c.Request.FormFile("chunk")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chunk file is required"})
			return
		}
		defer file.Close()
		if header.Size > maxChunkSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "chunk too large"})
			return
		}

		objectName := sessionID + "/chunk_" + strconv.Itoa(chunkIndex) + ".jpg"

		_, err = cfg.Minio.PutObject(context.Background(), cfg.BucketName, objectName, file, -1, minio.PutObjectOptions{
			ContentType: "image/jpeg",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload chunk"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRow(`SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			if err := LogAudit(cfg, projectID, "chunk_uploaded", "", map[string]interface{}{"session_id": sessionID, "chunk_index": chunkIndex}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"received":   true,
			"next_chunk": chunkIndex + 1,
		})
	}
}
