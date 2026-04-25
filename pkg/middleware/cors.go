package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS adds Cross-Origin Resource Sharing headers and handles OPTIONS preflight.
// It validates the incoming Origin header against an allowlist.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "https://app.chronoscope.io"
		}

		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if origin == allowedOrigin || strings.HasPrefix(allowedOrigin, "*") {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key, X-Chunk-Index")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
