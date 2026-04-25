package models

import (
	"time"
)

// Session represents a recorded user session.
type Session struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	UserID      string     `json:"user_id"`
	DurationMs  *int64     `json:"duration_ms"`
	VideoPath   *string    `json:"video_path"`
	EventCount  int        `json:"event_count"`
	ErrorCount  int        `json:"error_count"`
	Metadata    *string    `json:"metadata"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}
