package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/chronoscope/analytics/internal/config"
)

// HeatmapPoint represents a single coordinate with event density.
type HeatmapPoint struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Count int `json:"count"`
}

// GetHeatmap returns event density by (x, y) coordinates for a project.
func GetHeatmap(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, exists := c.Get("project_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
			return
		}

		rows, err := cfg.DB.Query(`
			SELECT e.x, e.y, COUNT(*) as count
			FROM events e
			JOIN sessions s ON e.session_id = s.id
			WHERE s.project_id = $1
			GROUP BY e.x, e.y
			ORDER BY count DESC
			LIMIT 100
		`, projectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query heatmap"})
			return
		}
		defer rows.Close()

		var points []HeatmapPoint
		for rows.Next() {
			var p HeatmapPoint
			if err := rows.Scan(&p.X, &p.Y, &p.Count); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan row"})
				return
			}
			points = append(points, p)
		}

		c.JSON(http.StatusOK, gin.H{
			"project_id": projectID,
			"points":     points,
		})
	}
}
