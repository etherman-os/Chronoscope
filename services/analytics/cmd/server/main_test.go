package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chronoscope/analytics/internal/config"
	"github.com/gin-gonic/gin"
)

func TestNewRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		ServerAddr: ":8081",
		DB:         db,
	}

	router := NewRouter(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/analytics/funnel", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing API key, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/analytics/heatmap", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing API key, got %d", w.Code)
	}
}
