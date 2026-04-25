package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

const (
	maxChunkSize  = 2 << 20 // 2 MiB
	maxChunkIndex = 10000
)

// isJPEG validates the magic bytes of a JPEG file.
func isJPEG(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	return data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
}

// UploadChunk handles multipart chunk uploads to MinIO.
func UploadChunk(cfg *config.Config) gin.HandlerFunc {
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

		chunkIndexStr := c.GetHeader("X-Chunk-Index")
		if chunkIndexStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-Index header is required"})
			return
		}

		chunkIndex, err := strconv.Atoi(chunkIndexStr)
		if err != nil || chunkIndex < 0 || chunkIndex >= maxChunkIndex {
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

		// Validate JPEG magic bytes
		magic := make([]byte, 3)
		if _, err := io.ReadFull(file, magic); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read chunk file"})
			return
		}
		if !isJPEG(magic) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chunk must be a valid JPEG image"})
			return
		}
		// Reset to beginning for upload
		if seeker, ok := file.(io.Seeker); ok {
			_, err = seeker.Seek(0, io.SeekStart)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to seek chunk file"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "chunk file does not support seeking"})
			return
		}

		objectName := sessionID + "/chunk_" + strconv.Itoa(chunkIndex) + ".jpg"

		_, err = cfg.Minio.PutObject(ctx, cfg.BucketName, objectName, file, header.Size, minio.PutObjectOptions{
			ContentType: "image/jpeg",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload chunk"})
			return
		}

		var projectID string
		if err := cfg.DB.QueryRowContext(ctx, `SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&projectID); err == nil {
			if err := LogAudit(ctx, cfg, projectID, "chunk_uploaded", "", map[string]interface{}{"session_id": sessionID, "chunk_index": chunkIndex}); err != nil {
				log.Printf("audit log failed: %v", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"received":   true,
			"next_chunk": chunkIndex + 1,
		})
	}
}
