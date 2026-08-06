package store

import (
	"encoding/json"
	"fmt"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) GetOrgSettings() (models.OrgSettings, error) {
	raw, err := s.GetSetting("org")
	if err != nil {
		return models.OrgSettings{}, nil
	}
	var cfg models.OrgSettings
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return models.OrgSettings{}, fmt.Errorf("parse org setting: %w", err)
	}
	return cfg, nil
}

func (s *Store) SaveOrgSettings(cfg models.OrgSettings) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting("org", string(raw))
}
