package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
	"github.com/chronoscope/ingestion/internal/handlers"
	"github.com/chronoscope/ingestion/internal/middleware"
	sharedmw "github.com/chronoscope/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func parseRateLimit() (int, time.Duration) {
	requests := 100
	if v := os.Getenv("RATE_LIMIT_REQUESTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			requests = n
		}
	}
	interval := time.Minute
	if v := os.Getenv("RATE_LIMIT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}
	return requests, interval
}

func NewRouter(cfg *config.Config) *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	router.Use(sharedmw.CORS())
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)
		c.Next()
	})

	v1 := router.Group("/v1")
	v1.Use(middleware.RateLimit(parseRateLimit()))
	v1.Use(sharedmw.APIKeyAuth(cfg.DB))

	v1.POST("/sessions/init", handlers.InitSession(cfg))
	v1.POST("/sessions/:id/chunks", handlers.UploadChunk(cfg))
	v1.POST("/sessions/:id/events", handlers.UploadEvents(cfg))
	v1.POST("/sessions/:id/complete", handlers.CompleteSession(cfg))
	v1.GET("/sessions", handlers.ListSessions(cfg))
	v1.GET("/sessions/:id", handlers.GetSession(cfg))

	v1.POST("/gdpr/export/:user_id", handlers.ExportUserData(cfg))
	v1.DELETE("/gdpr/delete/:user_id", handlers.DeleteUserData(cfg))
	v1.GET("/gdpr/audit-logs", handlers.ListAuditLogs(cfg))

	return router
}

func main() {
	cfg := config.Load()

	router := NewRouter(cfg)

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
