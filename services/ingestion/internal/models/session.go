package models

import (
	"database/sql"
	"time"
)

// Session represents a recorded user session.
type Session struct {
	ID          string
	ProjectID   string
	UserID      string
	DurationMs  sql.NullInt64
	VideoPath   sql.NullString
	EventCount  int
	ErrorCount  int
	Metadata    sql.NullString // JSON
	Status      string
	CreatedAt   time.Time
	CompletedAt sql.NullTime
}
