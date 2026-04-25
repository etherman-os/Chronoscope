package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("sets CORS headers", func(t *testing.T) {
		os.Unsetenv("CORS_ALLOWED_ORIGIN")
		mw := CORS()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		mw(c)

		if w.Header().Get("Access-Control-Allow-Origin") != "https://app.chronoscope.io" {
			t.Errorf("unexpected origin: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		os.Setenv("CORS_ALLOWED_ORIGIN", "https://example.com")
		defer os.Unsetenv("CORS_ALLOWED_ORIGIN")
		mw := CORS()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("OPTIONS", "/", nil)

		mw(c)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("unexpected origin: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
