package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
)

func LogAudit(cfg *config.Config, projectID string, action string, actor string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cfg.DB.ExecContext(ctx,
		`INSERT INTO audit_logs (project_id, action, actor, details, created_at) VALUES ($1, $2, $3, $4, $5)`,
		projectID,
		action,
		actor,
		detailsJSON,
		time.Now(),
	)
	return err
}
