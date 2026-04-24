package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/chronoscope/analytics/internal/config"
	"github.com/chronoscope/analytics/internal/handlers"
	"github.com/chronoscope/analytics/internal/middleware"
)

func main() {
	cfg := config.Load()
	router := gin.Default()
	router.Use(middleware.CORS())

	v1 := router.Group("/v1")
	v1.Use(middleware.APIKeyAuth(cfg.DB))
	{
		v1.GET("/analytics/heatmap", handlers.GetHeatmap(cfg))
		v1.GET("/analytics/funnel", handlers.GetFunnel(cfg))
		v1.GET("/analytics/sessions/stats", handlers.GetSessionStats(cfg))
	}

	log.Println("Analytics server starting on :8081")
	router.Run(":8081")
}
