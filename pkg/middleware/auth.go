package middleware

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func compareSHA256(hashHex, plain string) bool {
	h := sha256.Sum256([]byte(plain))
	return hashHex == hex.EncodeToString(h[:])
}

// APIKeyAuth validates the X-API-Key header against the projects table.
// Supports both bcrypt (preferred) and legacy SHA-256 hashes for migration.
func APIKeyAuth(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		rows, err := db.QueryContext(c.Request.Context(),
			"SELECT id, api_key_hash FROM projects WHERE api_key_hash IS NOT NULL",
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var projectID string
			var keyHash string
			if err := rows.Scan(&projectID, &keyHash); err != nil {
				continue
			}

			// Detect bcrypt format ($2a$, $2b$, $2y$)
			if strings.HasPrefix(keyHash, "$2") {
				if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(apiKey)); err == nil {
					c.Set("project_id", projectID)
					c.Next()
					return
				}
			} else {
				// Fallback to legacy SHA-256 (to be removed after migration)
				if compareSHA256(keyHash, apiKey) {
					c.Set("project_id", projectID)
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
	}
}
