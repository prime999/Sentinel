package store

import (
	"encoding/json"
	"fmt"

	"github.com/sentinel-monitoring/sentinel/internal/config"
)

func (s *Store) GetAuthConfig(fallback config.AuthConfig) (config.AuthConfig, error) {
	raw, err := s.GetSetting("auth")
	if err != nil {
		return fallback, nil
	}
	var cfg config.AuthConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return fallback, fmt.Errorf("parse auth setting: %w", err)
	}
	if cfg.Username == "" {
		return fallback, nil
	}
	if cfg.Password == "" {
		cfg.Password = fallback.Password
	}
	return cfg, nil
}

func (s *Store) SaveAuthConfig(cfg config.AuthConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting("auth", string(raw))
}
