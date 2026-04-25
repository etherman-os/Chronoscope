package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		rl := RateLimit(2, time.Minute)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.Header.Set("X-API-Key", "key-1")

		rl(c)
		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		rl := RateLimit(1, time.Minute)
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request, _ = http.NewRequest("GET", "/", nil)
		c1.Request.Header.Set("X-API-Key", "key-2")
		rl(c1)

		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request, _ = http.NewRequest("GET", "/", nil)
		c2.Request.Header.Set("X-API-Key", "key-2")
		rl(c2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w2.Code)
		}
	})

	t.Run("falls back to client IP", func(t *testing.T) {
		rl := RateLimit(1, time.Minute)
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request, _ = http.NewRequest("GET", "/", nil)
		c1.Request.RemoteAddr = "192.168.1.1:1234"
		rl(c1)

		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request, _ = http.NewRequest("GET", "/", nil)
		c2.Request.RemoteAddr = "192.168.1.1:1234"
		rl(c2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w2.Code)
		}
	})
}

func TestAllowRefill(t *testing.T) {
	rl := newRateLimiter(1, time.Millisecond, 1)
	if !rl.allow("key") {
		t.Error("expected first request to be allowed")
	}
	if rl.allow("key") {
		t.Error("expected second request to be blocked")
	}
	time.Sleep(2 * time.Millisecond)
	if !rl.allow("key") {
		t.Error("expected request after refill to be allowed")
	}
}

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Error("min(1,2) should be 1")
	}
	if min(3, 2) != 2 {
		t.Error("min(3,2) should be 2")
	}
}
