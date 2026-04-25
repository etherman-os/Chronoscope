package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds production-hardening HTTP headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' blob:; media-src 'self' blob:; img-src 'self' data: blob:")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		c.Next()
	}
}
