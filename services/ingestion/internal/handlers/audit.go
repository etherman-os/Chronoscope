package handlers

import (
	"encoding/json"
	"time"

	"github.com/chronoscope/ingestion/internal/config"
)

func LogAudit(cfg *config.Config, projectID string, action string, actor string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	_, err = cfg.DB.Exec(
		`INSERT INTO audit_logs (project_id, action, actor, details, created_at) VALUES ($1, $2, $3, $4, $5)`,
		projectID,
		action,
		actor,
		detailsJSON,
		time.Now(),
	)
	return err
}
