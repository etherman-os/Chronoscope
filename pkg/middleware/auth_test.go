package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid API key sets project_id and calls next", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		apiKey := "valid-api-key-123"
		projectID := "proj-123"
		hashHex := "$2a$10$gf.NT4vS/mClWzg0r1mAweLDwDNX1v5faKle.NamtfrmxTOveZ.O2"

		mock.ExpectQuery(`SELECT id, api_key_hash FROM projects WHERE api_key_hash IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "api_key_hash"}).AddRow(projectID, hashHex))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions", nil)
		c.Request.Header.Set("X-API-Key", apiKey)

		middleware := APIKeyAuth(db)
		middleware(c)

		if c.IsAborted() {
			t.Errorf("expected context not to be aborted, but it was")
		}
		pid, exists := c.Get("project_id")
		if !exists {
			t.Fatal("expected project_id to be set")
		}
		if pid != projectID {
			t.Errorf("expected project_id %q, got %q", projectID, pid)
		}
	})

	t.Run("missing header returns 401", func(t *testing.T) {
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions", nil)

		middleware := APIKeyAuth(db)
		middleware(c)

		if !c.IsAborted() {
			t.Error("expected context to be aborted")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid API key returns 401", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		apiKey := "invalid-key"
		// bcrypt hash of a different key
		hashHex := "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqQzBZN0UfGNEsKYGs5vTR6Q8WJkO"

		mock.ExpectQuery(`SELECT id, api_key_hash FROM projects WHERE api_key_hash IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "api_key_hash"}).AddRow("proj-123", hashHex))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions", nil)
		c.Request.Header.Set("X-API-Key", apiKey)

		middleware := APIKeyAuth(db)
		middleware(c)

		if !c.IsAborted() {
			t.Error("expected context to be aborted")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		apiKey := "some-key"

		mock.ExpectQuery(`SELECT id, api_key_hash FROM projects WHERE api_key_hash IS NOT NULL`).
			WillReturnError(sql.ErrConnDone)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/v1/sessions", nil)
		c.Request.Header.Set("X-API-Key", apiKey)

		middleware := APIKeyAuth(db)
		middleware(c)

		if !c.IsAborted() {
			t.Error("expected context to be aborted")
		}
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}
