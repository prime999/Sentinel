package store

import (
	"encoding/json"
	"fmt"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Store) DeleteSetting(key string) error {
	_, err := s.db.Exec(`DELETE FROM settings WHERE key = ?`, key)
	return err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (s *Store) GetSMTPConfig(fallback models.SMTPConfig) (models.SMTPConfig, error) {
	raw, err := s.GetSetting("smtp")
	if err != nil {
		fb := fallback
		// Fresh installs: treat configured fallback host as enabled.
		if fb.Host != "" {
			fb.Enabled = true
		}
		return fb, nil
	}
	var cfg models.SMTPConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return fallback, fmt.Errorf("parse smtp setting: %w", err)
	}
	if cfg.Host == "" {
		return fallback, nil
	}
	if cfg.Port == 0 {
		cfg.Port = fallback.Port
	}
	if cfg.Username == "" {
		cfg.Username = fallback.Username
	}
	if cfg.Password == "" {
		cfg.Password = fallback.Password
	}
	if cfg.From == "" {
		cfg.From = fallback.From
	}
	if cfg.AlertEmails == "" {
		cfg.AlertEmails = fallback.AlertEmails
	}
	// Backward compat: configs saved before "enabled" default to on when host is set.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &probe); err == nil {
		if _, ok := probe["enabled"]; !ok {
			cfg.Enabled = true
		}
	}
	return cfg, nil
}

func (s *Store) SaveSMTPConfig(cfg models.SMTPConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting("smtp", string(raw))
}
