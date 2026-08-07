package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/config"
	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	raw, _ := json.Marshal(tags)
	return string(raw)
}

func decodeTags(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}

func (s *Store) GetWebhooks() ([]models.WebhookConfig, error) {
	raw, err := s.GetSetting("webhooks")
	if err != nil {
		return nil, nil
	}
	var hooks []models.WebhookConfig
	if err := json.Unmarshal([]byte(raw), &hooks); err != nil {
		return nil, err
	}
	return hooks, nil
}

func (s *Store) SaveWebhooks(hooks []models.WebhookConfig) error {
	raw, err := json.Marshal(hooks)
	if err != nil {
		return err
	}
	return s.SetSetting("webhooks", string(raw))
}

func (s *Store) GetServerSettings(fallback config.ServerConfig) (models.ServerSettings, error) {
	raw, err := s.GetSetting("server")
	if err != nil {
		return models.ServerSettings{
			DashboardURL:  fallback.DashboardURL,
			RetentionDays: fallback.RetentionDays,
			Workers:       fallback.Workers,
		}, nil
	}
	var cfg models.ServerSettings
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return models.ServerSettings{}, err
	}
	if cfg.DashboardURL == "" {
		cfg.DashboardURL = fallback.DashboardURL
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = fallback.RetentionDays
	}
	if cfg.RetentionDays < 30 {
		cfg.RetentionDays = 30
	}
	if cfg.Workers == 0 {
		cfg.Workers = fallback.Workers
	}
	return cfg, nil
}

func (s *Store) SaveServerSettings(cfg models.ServerSettings) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting("server", string(raw))
}

func (s *Store) GetStatusPageConfig() (models.StatusPageConfig, error) {
	raw, err := s.GetSetting("status_page")
	if err != nil {
		return models.StatusPageConfig{
			Enabled:    false,
			Title:      "System Status",
			MonitorIDs: []string{},
		}, nil
	}
	var cfg models.StatusPageConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return models.StatusPageConfig{}, err
	}
	if cfg.Title == "" {
		cfg.Title = "System Status"
	}
	if cfg.MonitorIDs == nil {
		cfg.MonitorIDs = []string{}
	}
	return cfg, nil
}

func (s *Store) SaveStatusPageConfig(cfg models.StatusPageConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.SetSetting("status_page", string(raw))
}

func (s *Store) ListMaintenanceWindows() ([]models.MaintenanceWindow, error) {
	rows, err := s.db.Query(`
		SELECT id, name, monitor_id, starts_at, ends_at, created_at
		FROM maintenance_windows ORDER BY starts_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.MaintenanceWindow
	for rows.Next() {
		w, err := scanMaintenance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func (s *Store) CreateMaintenanceWindow(w *models.MaintenanceWindow) error {
	now := time.Now().UTC()
	w.ID = newID()
	w.CreatedAt = now
	var monitorID interface{}
	if w.MonitorID != "" {
		monitorID = w.MonitorID
	}
	_, err := s.db.Exec(`
		INSERT INTO maintenance_windows (id, name, monitor_id, starts_at, ends_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, monitorID, formatTime(w.StartsAt), formatTime(w.EndsAt), formatTime(now),
	)
	return err
}

func (s *Store) DeleteMaintenanceWindow(id string) error {
	_, err := s.db.Exec(`DELETE FROM maintenance_windows WHERE id = ?`, id)
	return err
}

func (s *Store) IsInMaintenance(monitorID string, at time.Time) (bool, error) {
	rows, err := s.db.Query(`
		SELECT monitor_id FROM maintenance_windows
		WHERE starts_at <= ? AND ends_at >= ?`, formatTime(at), formatTime(at))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var mid sql.NullString
		if err := rows.Scan(&mid); err != nil {
			return false, err
		}
		if !mid.Valid || mid.String == "" {
			return true, nil
		}
		if mid.String == monitorID {
			return true, nil
		}
	}
	return false, nil
}

func scanMaintenance(row interface{ Scan(dest ...any) error }) (*models.MaintenanceWindow, error) {
	var w models.MaintenanceWindow
	var monitorID sql.NullString
	var start, end, created string
	if err := row.Scan(&w.ID, &w.Name, &monitorID, &start, &end, &created); err != nil {
		return nil, err
	}
	if monitorID.Valid {
		w.MonitorID = monitorID.String
	}
	var err error
	w.StartsAt, err = parseTime(start)
	if err != nil {
		return nil, err
	}
	w.EndsAt, err = parseTime(end)
	if err != nil {
		return nil, err
	}
	w.CreatedAt, err = parseTime(created)
	return &w, err
}

func (s *Store) InsertAudit(actor, action, resource, detail string) error {
	_, err := s.db.Exec(`
		INSERT INTO audit_log (id, actor, action, resource, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		newID(), actor, action, resource, nullString(detail), formatTime(time.Now().UTC()),
	)
	return err
}

func (s *Store) ListAuditLog(limit int) ([]models.AuditEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT id, actor, action, resource, detail, created_at
		FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		var detail sql.NullString
		var created string
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &detail, &created); err != nil {
			return nil, err
		}
		e.Detail = nullableString(detail)
		e.CreatedAt, _ = parseTime(created)
		out = append(out, e)
	}
	return out, rows.Err()
}

func hashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) CreateAPIToken(userID, name, token string) (*models.APITokenCreated, error) {
	now := time.Now().UTC()
	id := newID()
	prefix := token
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	_, err := s.db.Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, name, hashAPIToken(token), prefix, formatTime(now),
	)
	if err != nil {
		return nil, err
	}
	return &models.APITokenCreated{
		APIToken: models.APIToken{ID: id, UserID: userID, Name: name, Prefix: prefix, CreatedAt: now},
		Token:    token,
	}, nil
}

func (s *Store) ListAPITokens(userID string) ([]models.APIToken, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, name, prefix, created_at, last_used_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.APIToken
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAPIToken(id, userID string) error {
	res, err := s.db.Exec(`DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token not found")
	}
	return nil
}

func (s *Store) GetUserByAPIToken(token string) (*models.User, error) {
	hash := hashAPIToken(token)
	var userID, tokenID string
	err := s.db.QueryRow(`
		SELECT id, user_id FROM api_tokens WHERE token_hash = ?`, hash,
	).Scan(&tokenID, &userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), tokenID)
	return s.GetUserByID(userID)
}

func scanAPIToken(row interface{ Scan(dest ...any) error }) (*models.APIToken, error) {
	var t models.APIToken
	var created, lastUsed sql.NullString
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &created, &lastUsed); err != nil {
		return nil, err
	}
	t.CreatedAt, _ = parseTime(nullableString(created))
	if lastUsed.Valid && lastUsed.String != "" {
		if ts, err := parseTime(lastUsed.String); err == nil {
			t.LastUsedAt = &ts
		}
	}
	return &t, nil
}

func (s *Store) GetMonitorByHeartbeatToken(token string) (*models.Monitor, error) {
	row := s.db.QueryRow(`
		SELECT `+monitorColumns+` FROM monitors WHERE heartbeat_token = ? AND type = ?`,
		token, string(models.MonitorHeartbeat))
	m, err := s.scanMonitorRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *Store) ListMonitorsFiltered(tag string) ([]models.MonitorListItem, error) {
	items, err := s.ListMonitors()
	if err != nil || tag == "" {
		return items, err
	}
	tag = strings.ToLower(strings.TrimSpace(tag))
	var filtered []models.MonitorListItem
	for _, item := range items {
		for _, t := range item.Tags {
			if strings.ToLower(t) == tag {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered, nil
}
