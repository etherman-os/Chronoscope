package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CORS adds Cross-Origin Resource Sharing headers and handles OPTIONS preflight.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "https://app.chronoscope.io"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key, X-Chunk-Index")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
