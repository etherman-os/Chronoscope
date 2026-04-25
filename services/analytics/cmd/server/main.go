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

	"github.com/chronoscope/analytics/internal/config"
	"github.com/chronoscope/analytics/internal/handlers"
	"github.com/chronoscope/analytics/internal/middleware"
	sharedmw "github.com/chronoscope/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	router.MaxMultipartMemory = 4 << 20 // 4 MiB

	router.Use(sharedmw.RequestID())
	router.Use(sharedmw.CORS())
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	})

	router.Use(sharedmw.SecurityHeaders())

	v1 := router.Group("/v1")
	requests, interval := parseRateLimit()
	v1.Use(middleware.RateLimit(requests, interval, cfg.Redis))
	v1.Use(sharedmw.APIKeyAuth(cfg.DB))
	{
		v1.GET("/analytics/heatmap", handlers.GetHeatmap(cfg))
		v1.GET("/analytics/funnel", handlers.GetFunnel(cfg))
		v1.GET("/analytics/sessions/stats", handlers.GetSessionStats(cfg))
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/healthz/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := cfg.DB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database unreachable"})
			return
		}
		if cfg.Redis != nil {
			if err := cfg.Redis.Ping(ctx).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "redis unreachable"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

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
	log.Println("Shutting down analytics server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Analytics server forced to shutdown: ", err)
	}
	log.Println("Analytics server exiting")
}
