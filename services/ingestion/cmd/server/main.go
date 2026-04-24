package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/handlers"
	"github.com/chronoscope/ingestion/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	router := gin.Default()

	router.Use(middleware.CORS())

	v1 := router.Group("/v1")
	v1.Use(middleware.APIKeyAuth(cfg.DB))

	v1.POST("/sessions/init", handlers.InitSession(cfg))
	v1.POST("/sessions/:id/chunks", handlers.UploadChunk(cfg))
	v1.POST("/sessions/:id/events", handlers.UploadEvents(cfg))
	v1.POST("/sessions/:id/complete", handlers.CompleteSession(cfg))
	v1.GET("/sessions", handlers.ListSessions(cfg))
	v1.GET("/sessions/:id", handlers.GetSession(cfg))

	v1.POST("/gdpr/export/:user_id", handlers.ExportUserData(cfg))
	v1.DELETE("/gdpr/delete/:user_id", handlers.DeleteUserData(cfg))
	v1.GET("/gdpr/audit-logs", handlers.ListAuditLogs(cfg))

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
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
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}
	log.Println("Server exiting")
}
