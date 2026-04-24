package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	v1.Use(middleware.RateLimit(100, time.Minute))
	v1.Use(middleware.APIKeyAuth(cfg.DB))
	{
		v1.GET("/analytics/heatmap", handlers.GetHeatmap(cfg))
		v1.GET("/analytics/funnel", handlers.GetFunnel(cfg))
		v1.GET("/analytics/sessions/stats", handlers.GetSessionStats(cfg))
	}

	srv := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down analytics server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Analytics server forced to shutdown: ", err)
	}
	log.Println("Analytics server exiting")
}
