package models

import "time"

type LogSourceType string

const (
	LogSourceFile    LogSourceType = "file"
	LogSourceJournal LogSourceType = "journal" // phase 2
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelCrit  LogLevel = "CRITICAL"
)

type LogSourceHealth string

const (
	LogHealthOK               LogSourceHealth = "ok"
	LogHealthRejectedPolicy   LogSourceHealth = "rejected_policy"
	LogHealthPermissionDenied LogSourceHealth = "permission_denied"
	LogHealthFileMissing      LogSourceHealth = "file_missing"
	LogHealthRotated          LogSourceHealth = "rotated"
	LogHealthLagging          LogSourceHealth = "lagging"
	LogHealthRateLimited      LogSourceHealth = "rate_limited"
)

// LogSource is configured on a host (metadata in main DB).
type LogSource struct {
	ID             string         `json:"id"`
	HostID         string         `json:"host_id"`
	Name           string         `json:"name"`
	Type           LogSourceType  `json:"type"`
	Path           string         `json:"path"`
	Tags           []string       `json:"tags"`
	Enabled        bool           `json:"enabled"`
	MinLevel       LogLevel       `json:"min_level"`
	IncludePatterns []string      `json:"include_patterns"`
	Health         LogSourceHealth `json:"health"`
	HealthDetail   string         `json:"health_detail,omitempty"`
	DroppedLines   int64          `json:"dropped_lines"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// LogAlertRule fires when pattern matches exceed threshold in a window.
type LogAlertRule struct {
	ID             string    `json:"id"`
	HostID         string    `json:"host_id"`
	SourceID       string    `json:"source_id,omitempty"` // empty = all sources on host
	Name           string    `json:"name"`
	Pattern        string    `json:"pattern"`
	Threshold      int       `json:"threshold"`
	WindowSeconds  int       `json:"window_seconds"`
	Severity       string    `json:"severity"` // warning | critical
	Enabled        bool      `json:"enabled"`
	NotifyEmail    bool      `json:"notify_email"`
	NotifySlack    bool      `json:"notify_slack"`
	NotifyWebhooks bool      `json:"notify_webhooks"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LogEvent is a matched line stored in sentinel-logs.db.
type LogEvent struct {
	ID          string    `json:"id"`
	HostID      string    `json:"host_id"`
	SourceID    string    `json:"source_id"`
	Timestamp   time.Time `json:"timestamp"`
	Level       LogLevel  `json:"level"`
	Message     string    `json:"message"`
	Fingerprint string    `json:"fingerprint"`
	Metadata    string    `json:"metadata,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
}

// LogVolumeBucket is a 1-minute count in sentinel-logs.db.
type LogVolumeBucket struct {
	HostID      string    `json:"host_id"`
	SourceID    string    `json:"source_id"`
	BucketStart time.Time `json:"bucket_start"`
	Level       LogLevel  `json:"level"`
	Count       int64     `json:"count"`
}

// LogSettings are first-class platform caps (stored in main DB settings).
type LogSettings struct {
	RetentionDays      int   `json:"retention_days"`
	VolumeRetentionDays int  `json:"volume_retention_days"`
	MaxDBSizeBytes     int64 `json:"max_db_size_bytes"`
	MaxEventsPerSecHost int  `json:"max_events_per_sec_host"`
	MaxEventsPerSecGlobal int `json:"max_events_per_sec_global"`
	MaxEventSizeBytes  int   `json:"max_event_size_bytes"`
	MaxBatchEvents     int   `json:"max_batch_events"`
}

func DefaultLogSettings() LogSettings {
	return LogSettings{
		RetentionDays:         7,
		VolumeRetentionDays:   30,
		MaxDBSizeBytes:        2 * 1024 * 1024 * 1024, // 2 GB
		MaxEventsPerSecHost:   100,
		MaxEventsPerSecGlobal: 1000,
		MaxEventSizeBytes:     32 * 1024,
		MaxBatchEvents:        200,
	}
}

// AgentLogSource is pushed to the agent in HostAgentConfig.
type AgentLogSource struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type,omitempty"` // file | journal
	Path            string   `json:"path"`           // file path or journal unit
	Enabled         bool     `json:"enabled"`
	MinLevel        string   `json:"min_level"`
	IncludePatterns []string `json:"include_patterns"`
	MaxEventBytes   int      `json:"max_event_bytes"`
	MaxEventsPerSec int      `json:"max_events_per_sec"`
}

// AgentTailLease asks the agent to stream lines for a short live-tail window.
type AgentTailLease struct {
	SourceID string `json:"source_id"`
	Until    string `json:"until"` // RFC3339 UTC
}

// LogIngestBatch is posted by the agent.
type LogIngestBatch struct {
	HostID  string            `json:"host_id,omitempty"`
	Events  []LogIngestEvent  `json:"events"`
	Volumes []LogIngestVolume `json:"volumes,omitempty"`
	Health  []LogIngestHealth `json:"health,omitempty"`
}

type LogIngestEvent struct {
	SourceID    string `json:"source_id"`
	Timestamp   string `json:"timestamp"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	Fingerprint string `json:"fingerprint"`
	Metadata    string `json:"metadata,omitempty"`
}

type LogIngestVolume struct {
	SourceID    string `json:"source_id"`
	BucketStart string `json:"bucket_start"`
	Level       string `json:"level"`
	Count       int64  `json:"count"`
}

type LogIngestHealth struct {
	SourceID     string `json:"source_id"`
	Health       string `json:"health"`
	Detail       string `json:"detail,omitempty"`
	DroppedLines int64  `json:"dropped_lines"`
}

// LogSourceTemplate helps admins pick safe paths.
type LogSourceTemplate struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	Tags            []string `json:"tags"`
	MinLevel        LogLevel `json:"min_level"`
	IncludePatterns []string `json:"include_patterns"`
}

func DefaultLogSourceTemplates() []LogSourceTemplate {
	return []LogSourceTemplate{
		{
			ID: "nginx-error", Name: "Nginx Error Log", Path: "/var/log/nginx/error.log",
			Tags: []string{"nginx"}, MinLevel: LogLevelWarn,
			IncludePatterns: []string{"ERROR", "CRITICAL", "emerg", "alert", "crit"},
		},
		{
			ID: "nginx-access", Name: "Nginx Access Log", Path: "/var/log/nginx/access.log",
			Tags: []string{"nginx"}, MinLevel: LogLevelWarn,
			IncludePatterns: []string{` [45]\d\d `, "error", "timeout"},
		},
		{
			ID: "laravel", Name: "Laravel Log", Path: "/var/www/*/storage/logs/*.log",
			Tags: []string{"laravel", "application"}, MinLevel: LogLevelWarn,
			IncludePatterns: []string{"ERROR", "CRITICAL", "Exception", "timeout"},
		},
		{
			ID: "php-fpm", Name: "PHP-FPM Log", Path: "/var/log/php*-fpm.log",
			Tags: []string{"php"}, MinLevel: LogLevelWarn,
			IncludePatterns: []string{"ERROR", "WARNING", "ALERT"},
		},
	}
}
