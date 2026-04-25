package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chronoscope/ingestion/internal/config"
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
		ServerAddr: ":8080",
		DB:         db,
		Minio:      nil,
		BucketName: "test",
	}

	router := NewRouter(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing API key, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/v1/sessions/init", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing API key, got %d", w.Code)
	}
}
