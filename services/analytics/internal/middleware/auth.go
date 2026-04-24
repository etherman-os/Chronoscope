package middleware

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth validates the X-API-Key header against the projects table.
func APIKeyAuth(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		var projectID string
		err := db.QueryRow("SELECT id FROM projects WHERE api_key_hash = $1", apiKey).Scan(&projectID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Set("project_id", projectID)
		c.Next()
	}
}
