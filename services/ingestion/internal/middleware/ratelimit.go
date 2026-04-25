package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter implements a simple per-key token bucket rate limiter
// with automatic bucket cleanup to prevent memory leaks.
type rateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // tokens added per interval
	interval time.Duration // interval between token adds
	burst    int           // max bucket size
	ttl      time.Duration // bucket lifetime after last access
}

type bucket struct {
	tokens    int
	lastCheck time.Time
}

func newRateLimiter(rate int, interval time.Duration, burst int) *rateLimiter {
	rl := &rateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
		ttl:      1 * time.Hour,
	}
	// Start background cleanup goroutine
	go rl.cleanup()
	return rl
}

// cleanup removes stale buckets every 10 minutes to prevent memory leaks.
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastCheck) > rl.ttl {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, lastCheck: time.Now()}
		return true
	}

	now := time.Now()
	elapsed := now.Sub(b.lastCheck)
	refill := int(elapsed / rl.interval)

	if refill > 0 {
		b.tokens = min(b.tokens+refill, rl.burst)
		b.lastCheck = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimit creates a middleware that limits requests per API key.
func RateLimit(requests int, interval time.Duration) gin.HandlerFunc {
	rl := newRateLimiter(requests, interval, requests)
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			key = c.ClientIP()
		}
		if !rl.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
