package config

import (
	"fmt"
	"os"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	SMTP     models.SMTPConfig `yaml:"smtp"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Listen        string `yaml:"listen"`
	Workers       int    `yaml:"workers"`
	RetentionDays int    `yaml:"retention_days"`
	DashboardURL  string `yaml:"dashboard_url"`
}

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = "0.0.0.0:8082"
	}
	if c.Server.Workers == 0 {
		c.Server.Workers = 10
	}
	if c.Server.RetentionDays == 0 {
		c.Server.RetentionDays = 30
	}
	if c.Server.RetentionDays < 30 {
		c.Server.RetentionDays = 30
	}
	if c.Server.DashboardURL == "" {
		c.Server.DashboardURL = "http://localhost:8082"
	}
	if c.Auth.Username == "" {
		c.Auth.Username = "admin"
	}
	if c.Database.Path == "" {
		c.Database.Path = "./data/sentinel.db"
	}
	if c.SMTP.Port == 0 {
		c.SMTP.Port = 587
	}
}
