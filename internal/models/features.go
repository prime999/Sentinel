package models

import "time"

type WebhookConfig struct {
	URL      string   `json:"url"`
	Enabled  bool     `json:"enabled"`
	Events   []string `json:"events"`
}

type MaintenanceWindow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MonitorID string    `json:"monitor_id,omitempty"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditEntry struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type APIToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type APITokenCreated struct {
	APIToken
	Token string `json:"token"`
}

type ServerSettings struct {
	DashboardURL  string `json:"dashboard_url"`
	RetentionDays int    `json:"retention_days"`
	Workers       int    `json:"workers"`
}

type StatusPageConfig struct {
	Enabled    bool     `json:"enabled"`
	Title      string   `json:"title"`
	MonitorIDs []string `json:"monitor_ids"`
}

type PublicMonitorStatus struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	LastCheck  string `json:"last_checked_at,omitempty"`
	URL        string `json:"url,omitempty"`
}

type PublicStatusResponse struct {
	Title    string                `json:"title"`
	Monitors []PublicMonitorStatus `json:"monitors"`
}

type IncidentListItem struct {
	Incident
	MonitorName string `json:"monitor_name"`
}

type HeartbeatConfig struct {
	Token        string `json:"token"`
	GraceSeconds int    `json:"grace_seconds"`
}
