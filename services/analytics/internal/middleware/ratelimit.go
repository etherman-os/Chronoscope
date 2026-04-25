package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end
if current > limit then
    return 0
end
return 1
`)

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int
	interval time.Duration
	burst    int
	ttl      time.Duration
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
	go rl.cleanup()
	return rl
}

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
		b.tokens = minInt(b.tokens+refill, rl.burst)
		b.lastCheck = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimit creates a middleware that limits requests per API key using Redis.
// Falls back to in-memory limiting if redisClient is nil (not recommended for production).
func RateLimit(requests int, interval time.Duration, redisClient *redis.Client) gin.HandlerFunc {
	if redisClient == nil {
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

	windowSecs := int(interval.Seconds())
	if windowSecs < 1 {
		windowSecs = 60
	}

	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			key = c.ClientIP()
		}
		redisKey := "ratelimit:" + key

		allowed, err := rateLimitScript.Run(c.Request.Context(), redisClient, []string{redisKey}, requests, windowSecs).Bool()
		if err != nil || !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
