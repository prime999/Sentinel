package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

const logSettingsKey = "logs"

func (s *Store) GetLogSettings() (models.LogSettings, error) {
	def := models.DefaultLogSettings()
	raw, err := s.GetSetting(logSettingsKey)
	if err != nil {
		return def, nil
	}
	var cfg models.LogSettings
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return def, nil
	}
	return normalizeLogSettings(cfg), nil
}

func (s *Store) SaveLogSettings(cfg models.LogSettings) error {
	cfg = normalizeLogSettings(cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting(logSettingsKey, string(raw))
}

func normalizeLogSettings(cfg models.LogSettings) models.LogSettings {
	def := models.DefaultLogSettings()
	if cfg.RetentionDays < 1 {
		cfg.RetentionDays = def.RetentionDays
	}
	if cfg.VolumeRetentionDays < 1 {
		cfg.VolumeRetentionDays = def.VolumeRetentionDays
	}
	if cfg.MaxDBSizeBytes < 64*1024*1024 {
		cfg.MaxDBSizeBytes = def.MaxDBSizeBytes
	}
	if cfg.MaxEventsPerSecHost < 1 {
		cfg.MaxEventsPerSecHost = def.MaxEventsPerSecHost
	}
	if cfg.MaxEventsPerSecGlobal < 1 {
		cfg.MaxEventsPerSecGlobal = def.MaxEventsPerSecGlobal
	}
	if cfg.MaxEventSizeBytes < 1024 {
		cfg.MaxEventSizeBytes = def.MaxEventSizeBytes
	}
	if cfg.MaxBatchEvents < 10 {
		cfg.MaxBatchEvents = def.MaxBatchEvents
	}
	if cfg.MaxBatchEvents > 500 {
		cfg.MaxBatchEvents = 500
	}
	return cfg
}

func (s *Store) ListLogSources(hostID string) ([]models.LogSource, error) {
	rows, err := s.db.Query(`
		SELECT id, host_id, name, type, path, tags, enabled, min_level, include_patterns,
		       health, health_detail, dropped_lines, created_at, updated_at
		FROM log_sources WHERE host_id = ? ORDER BY name ASC`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogSource
	for rows.Next() {
		src, err := scanLogSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *src)
	}
	if out == nil {
		out = []models.LogSource{}
	}
	return out, rows.Err()
}

func (s *Store) ListEnabledLogSources(hostID string) ([]models.LogSource, error) {
	all, err := s.ListLogSources(hostID)
	if err != nil {
		return nil, err
	}
	var out []models.LogSource
	for _, src := range all {
		if src.Enabled {
			out = append(out, src)
		}
	}
	if out == nil {
		out = []models.LogSource{}
	}
	return out, nil
}

func (s *Store) GetLogSource(id string) (*models.LogSource, error) {
	row := s.db.QueryRow(`
		SELECT id, host_id, name, type, path, tags, enabled, min_level, include_patterns,
		       health, health_detail, dropped_lines, created_at, updated_at
		FROM log_sources WHERE id = ?`, id)
	src, err := scanLogSource(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return src, err
}

func (s *Store) CreateLogSource(src *models.LogSource) error {
	if src.ID == "" {
		src.ID = newID()
	}
	now := time.Now().UTC()
	src.CreatedAt = now
	src.UpdatedAt = now
	if src.Type == "" {
		src.Type = models.LogSourceFile
	}
	if src.MinLevel == "" {
		src.MinLevel = models.LogLevelWarn
	}
	if src.Health == "" {
		src.Health = models.LogHealthOK
	}
	if src.Tags == nil {
		src.Tags = []string{}
	}
	if src.IncludePatterns == nil {
		src.IncludePatterns = []string{}
	}
	_, err := s.db.Exec(`
		INSERT INTO log_sources (
			id, host_id, name, type, path, tags, enabled, min_level, include_patterns,
			health, health_detail, dropped_lines, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		src.ID, src.HostID, src.Name, string(src.Type), src.Path, encodeTags(src.Tags),
		boolToInt(src.Enabled), string(src.MinLevel), encodeTags(src.IncludePatterns),
		string(src.Health), src.HealthDetail, src.DroppedLines,
		formatTime(src.CreatedAt), formatTime(src.UpdatedAt),
	)
	return err
}

func (s *Store) UpdateLogSource(src *models.LogSource) error {
	src.UpdatedAt = time.Now().UTC()
	if src.Tags == nil {
		src.Tags = []string{}
	}
	if src.IncludePatterns == nil {
		src.IncludePatterns = []string{}
	}
	_, err := s.db.Exec(`
		UPDATE log_sources SET
			name=?, type=?, path=?, tags=?, enabled=?, min_level=?, include_patterns=?,
			health=?, health_detail=?, dropped_lines=?, updated_at=?
		WHERE id=?`,
		src.Name, string(src.Type), src.Path, encodeTags(src.Tags), boolToInt(src.Enabled),
		string(src.MinLevel), encodeTags(src.IncludePatterns), string(src.Health),
		src.HealthDetail, src.DroppedLines, formatTime(src.UpdatedAt), src.ID,
	)
	return err
}

func (s *Store) UpdateLogSourceHealth(id string, health models.LogSourceHealth, detail string, dropped int64) error {
	_, err := s.db.Exec(`
		UPDATE log_sources SET health=?, health_detail=?, dropped_lines=?, updated_at=?
		WHERE id=?`,
		string(health), detail, dropped, formatTime(time.Now().UTC()), id,
	)
	return err
}

func (s *Store) DeleteLogSource(id string) error {
	_, err := s.db.Exec(`DELETE FROM log_sources WHERE id = ?`, id)
	return err
}

func (s *Store) CountLogSourcesByHost(hostID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM log_sources WHERE host_id = ?`, hostID).Scan(&n)
	return n, err
}

func (s *Store) ListLogAlertRules(hostID string) ([]models.LogAlertRule, error) {
	rows, err := s.db.Query(`
		SELECT id, host_id, COALESCE(source_id,''), name, pattern, threshold, window_seconds,
		       severity, enabled, notify_email, notify_slack, notify_webhooks, created_at, updated_at
		FROM log_alert_rules WHERE host_id = ? ORDER BY name ASC`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogAlertRule
	for rows.Next() {
		r, err := scanLogAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	if out == nil {
		out = []models.LogAlertRule{}
	}
	return out, rows.Err()
}

func (s *Store) ListEnabledLogAlertRules(hostID string) ([]models.LogAlertRule, error) {
	all, err := s.ListLogAlertRules(hostID)
	if err != nil {
		return nil, err
	}
	var out []models.LogAlertRule
	for _, r := range all {
		if r.Enabled {
			out = append(out, r)
		}
	}
	if out == nil {
		out = []models.LogAlertRule{}
	}
	return out, nil
}

func (s *Store) GetLogAlertRule(id string) (*models.LogAlertRule, error) {
	row := s.db.QueryRow(`
		SELECT id, host_id, COALESCE(source_id,''), name, pattern, threshold, window_seconds,
		       severity, enabled, notify_email, notify_slack, notify_webhooks, created_at, updated_at
		FROM log_alert_rules WHERE id = ?`, id)
	r, err := scanLogAlertRule(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (s *Store) CreateLogAlertRule(r *models.LogAlertRule) error {
	if r.ID == "" {
		r.ID = newID()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Threshold < 1 {
		r.Threshold = 5
	}
	if r.WindowSeconds < 60 {
		r.WindowSeconds = 300
	}
	if r.Severity == "" {
		r.Severity = "warning"
	}
	var sourceID any
	if r.SourceID != "" {
		sourceID = r.SourceID
	}
	_, err := s.db.Exec(`
		INSERT INTO log_alert_rules (
			id, host_id, source_id, name, pattern, threshold, window_seconds, severity,
			enabled, notify_email, notify_slack, notify_webhooks, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.HostID, sourceID, r.Name, r.Pattern, r.Threshold, r.WindowSeconds, r.Severity,
		boolToInt(r.Enabled), boolToInt(r.NotifyEmail), boolToInt(r.NotifySlack), boolToInt(r.NotifyWebhooks),
		formatTime(r.CreatedAt), formatTime(r.UpdatedAt),
	)
	return err
}

func (s *Store) UpdateLogAlertRule(r *models.LogAlertRule) error {
	r.UpdatedAt = time.Now().UTC()
	var sourceID any
	if r.SourceID != "" {
		sourceID = r.SourceID
	}
	_, err := s.db.Exec(`
		UPDATE log_alert_rules SET
			source_id=?, name=?, pattern=?, threshold=?, window_seconds=?, severity=?,
			enabled=?, notify_email=?, notify_slack=?, notify_webhooks=?, updated_at=?
		WHERE id=?`,
		sourceID, r.Name, r.Pattern, r.Threshold, r.WindowSeconds, r.Severity,
		boolToInt(r.Enabled), boolToInt(r.NotifyEmail), boolToInt(r.NotifySlack), boolToInt(r.NotifyWebhooks),
		formatTime(r.UpdatedAt), r.ID,
	)
	return err
}

func (s *Store) DeleteLogAlertRule(id string) error {
	_, err := s.db.Exec(`DELETE FROM log_alert_rules WHERE id = ?`, id)
	return err
}

func (s *Store) GetOpenLogRuleIncident(hostID, ruleID string) (*models.Incident, error) {
	needle := `"log_rule_id":"` + ruleID + `"`
	row := s.db.QueryRow(`
		SELECT id, monitor_id, type, message, started_at, resolved_at, acknowledged_at, acknowledged_by, details
		FROM incidents
		WHERE monitor_id = ? AND type = ? AND resolved_at IS NULL AND details LIKE ?
		ORDER BY started_at DESC LIMIT 1`,
		hostID, string(models.IncidentHostLog), "%"+needle+"%",
	)
	return scanIncidentWithDetails(row)
}

func (s *Store) ResolveOpenLogRuleIncident(hostID, ruleID string, resolvedAt time.Time) error {
	needle := `"log_rule_id":"` + ruleID + `"`
	_, err := s.db.Exec(`
		UPDATE incidents SET resolved_at = ?
		WHERE monitor_id = ? AND type = ? AND resolved_at IS NULL AND details LIKE ?`,
		formatTime(resolvedAt), hostID, string(models.IncidentHostLog), "%"+needle+"%",
	)
	return err
}

func (s *Store) ListLogSourcesForHosts(hostIDs []string) ([]models.LogSource, error) {
	if len(hostIDs) == 0 {
		return []models.LogSource{}, nil
	}
	args := make([]any, len(hostIDs))
	for i, id := range hostIDs {
		args[i] = id
	}
	rows, err := s.db.Query(`
		SELECT id, host_id, name, type, path, tags, enabled, min_level, include_patterns,
		       health, health_detail, dropped_lines, created_at, updated_at
		FROM log_sources WHERE host_id IN (`+sqlPlaceholders(len(hostIDs))+`)
		ORDER BY host_id, name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogSource
	for rows.Next() {
		src, err := scanLogSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *src)
	}
	if out == nil {
		out = []models.LogSource{}
	}
	return out, rows.Err()
}

func (s *Store) ListLogAlertRulesForHosts(hostIDs []string) ([]models.LogAlertRule, error) {
	if len(hostIDs) == 0 {
		return []models.LogAlertRule{}, nil
	}
	args := make([]any, len(hostIDs))
	for i, id := range hostIDs {
		args[i] = id
	}
	rows, err := s.db.Query(`
		SELECT id, host_id, COALESCE(source_id,''), name, pattern, threshold, window_seconds,
		       severity, enabled, notify_email, notify_slack, notify_webhooks, created_at, updated_at
		FROM log_alert_rules WHERE host_id IN (`+sqlPlaceholders(len(hostIDs))+`)
		ORDER BY host_id, name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogAlertRule
	for rows.Next() {
		r, err := scanLogAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	if out == nil {
		out = []models.LogAlertRule{}
	}
	return out, rows.Err()
}

func scanLogSource(row interface {
	Scan(dest ...any) error
}) (*models.LogSource, error) {
	var src models.LogSource
	var typ, minLevel, health, tagsRaw, patternsRaw, created, updated string
	var enabled int
	if err := row.Scan(
		&src.ID, &src.HostID, &src.Name, &typ, &src.Path, &tagsRaw, &enabled, &minLevel, &patternsRaw,
		&health, &src.HealthDetail, &src.DroppedLines, &created, &updated,
	); err != nil {
		return nil, err
	}
	src.Type = models.LogSourceType(typ)
	src.MinLevel = models.LogLevel(minLevel)
	src.Health = models.LogSourceHealth(health)
	src.Enabled = enabled != 0
	src.Tags = decodeTags(tagsRaw)
	src.IncludePatterns = decodeTags(patternsRaw)
	if t, err := parseTime(created); err == nil {
		src.CreatedAt = t
	}
	if t, err := parseTime(updated); err == nil {
		src.UpdatedAt = t
	}
	return &src, nil
}

func scanLogAlertRule(row interface {
	Scan(dest ...any) error
}) (*models.LogAlertRule, error) {
	var r models.LogAlertRule
	var created, updated string
	var enabled, notifyEmail, notifySlack, notifyWebhooks int
	if err := row.Scan(
		&r.ID, &r.HostID, &r.SourceID, &r.Name, &r.Pattern, &r.Threshold, &r.WindowSeconds,
		&r.Severity, &enabled, &notifyEmail, &notifySlack, &notifyWebhooks, &created, &updated,
	); err != nil {
		return nil, err
	}
	r.Enabled = enabled != 0
	r.NotifyEmail = notifyEmail != 0
	r.NotifySlack = notifySlack != 0
	r.NotifyWebhooks = notifyWebhooks != 0
	if t, err := parseTime(created); err == nil {
		r.CreatedAt = t
	}
	if t, err := parseTime(updated); err == nil {
		r.UpdatedAt = t
	}
	return &r, nil
}

func scanIncidentWithDetails(row interface {
	Scan(dest ...any) error
}) (*models.Incident, error) {
	var inc models.Incident
	var incType, started string
	var resolved, ackAt sql.NullString
	var details sql.NullString
	if err := row.Scan(
		&inc.ID, &inc.MonitorID, &incType, &inc.Message, &started, &resolved, &ackAt, &inc.AcknowledgedBy, &details,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	inc.Type = models.IncidentType(incType)
	if t, err := parseTime(started); err == nil {
		inc.StartedAt = t
	}
	if resolved.Valid && resolved.String != "" {
		if t, err := parseTime(resolved.String); err == nil {
			inc.ResolvedAt = &t
		}
	}
	if ackAt.Valid && ackAt.String != "" {
		if t, err := parseTime(ackAt.String); err == nil {
			inc.AcknowledgedAt = &t
		}
	}
	if details.Valid {
		inc.Details = details.String
		inc.ErrorPage = decodeIncidentErrorPage(details.String)
	}
	return &inc, nil
}

// AgentLogSources builds the config payload for a host.
func (s *Store) AgentLogSources(hostID string, settings models.LogSettings) ([]models.AgentLogSource, error) {
	sources, err := s.ListEnabledLogSources(hostID)
	if err != nil {
		return nil, err
	}
	out := make([]models.AgentLogSource, 0, len(sources))
	for _, src := range sources {
		if src.Type != models.LogSourceFile && src.Type != models.LogSourceJournal {
			continue
		}
		out = append(out, models.AgentLogSource{
			ID:              src.ID,
			Name:            src.Name,
			Path:            src.Path,
			Enabled:         src.Enabled,
			MinLevel:        string(src.MinLevel),
			IncludePatterns: src.IncludePatterns,
			MaxEventBytes:   settings.MaxEventSizeBytes,
			MaxEventsPerSec: settings.MaxEventsPerSecHost,
			Type:            string(src.Type),
		})
	}
	return out, nil
}

func LogRuleDetailsJSON(ruleID string) string {
	return fmt.Sprintf(`{"log_rule_id":%q}`, ruleID)
}

func ParseLogRuleID(details string) string {
	const prefix = `"log_rule_id":"`
	i := strings.Index(details, prefix)
	if i < 0 {
		return ""
	}
	rest := details[i+len(prefix):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}
