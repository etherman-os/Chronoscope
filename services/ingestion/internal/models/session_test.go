package models

import (
	"database/sql"
	"testing"
	"time"
)

func TestSessionStruct(t *testing.T) {
	s := Session{
		ID:          "sess-1",
		ProjectID:   "proj-1",
		UserID:      "user-1",
		DurationMs:  sql.NullInt64{Int64: 1000, Valid: true},
		VideoPath:   sql.NullString{String: "/tmp/video.mp4", Valid: true},
		EventCount:  10,
		ErrorCount:  1,
		Metadata:    sql.NullString{String: `{"browser":"chrome"}`, Valid: true},
		Status:      "completed",
		CreatedAt:   time.Now(),
		CompletedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}
	if s.ID != "sess-1" {
		t.Error("ID mismatch")
	}
	if s.ProjectID != "proj-1" {
		t.Error("ProjectID mismatch")
	}
	if s.UserID != "user-1" {
		t.Error("UserID mismatch")
	}
	if !s.DurationMs.Valid || s.DurationMs.Int64 != 1000 {
		t.Error("DurationMs mismatch")
	}
	if s.EventCount != 10 {
		t.Error("EventCount mismatch")
	}
}
