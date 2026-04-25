package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionStruct(t *testing.T) {
	durationMs := int64(1000)
	videoPath := "/tmp/video.mp4"
	metadata := `{"browser":"chrome"}`
	completedAt := time.Now()
	s := Session{
		ID:          "sess-1",
		ProjectID:   "proj-1",
		UserID:      "user-1",
		DurationMs:  &durationMs,
		VideoPath:   &videoPath,
		EventCount:  10,
		ErrorCount:  1,
		Metadata:    &metadata,
		Status:      "completed",
		CreatedAt:   time.Now(),
		CompletedAt: &completedAt,
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
	if s.DurationMs == nil || *s.DurationMs != 1000 {
		t.Error("DurationMs mismatch")
	}
	if s.EventCount != 10 {
		t.Error("EventCount mismatch")
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := m["id"]; !ok {
		t.Errorf("expected 'id' key in JSON, got %s", string(b))
	}
	if _, ok := m["ID"]; ok {
		t.Errorf("unexpected 'ID' key in JSON, got %s", string(b))
	}
}
