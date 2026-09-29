package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

// ListMonitorsForExport returns full monitor rows (including secrets) for backup export.
// tenantID empty means all monitors; otherwise scoped to that tenant.
func (s *Store) ListMonitorsForExport(tenantID string) ([]models.Monitor, error) {
	q := `SELECT ` + monitorColumns + ` FROM monitors`
	var args []any
	if tenantID != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, tenantID)
	}
	q += ` ORDER BY created_at ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Monitor
	for rows.Next() {
		m, err := s.scanMonitorRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// GetMonitorsByIDs loads monitors by id for import preview and conflict detection.
func (s *Store) GetMonitorsByIDs(ids []string) (map[string]models.Monitor, error) {
	out := make(map[string]models.Monitor, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`SELECT `+monitorColumns+` FROM monitors WHERE id IN (`+sqlPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		m, err := s.scanMonitorRow(rows)
		if err != nil {
			return nil, err
		}
		out[m.ID] = *m
	}
	return out, rows.Err()
}

func prepareMonitorImport(m *models.Monitor) {
	now := time.Now().UTC()
	if strings.TrimSpace(m.ID) == "" {
		m.ID = newID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	if m.Type == "" {
		m.Type = models.MonitorHTTP
	}
	if m.Method == "" {
		m.Method = "GET"
	}
	if m.IntervalSeconds < 30 {
		if m.IntervalSeconds == 0 {
			m.IntervalSeconds = 60
		} else {
			m.IntervalSeconds = 30
		}
	}
	if m.TimeoutMs == 0 {
		m.TimeoutMs = 10000
	}
	if m.SlowThresholdMs == 0 {
		m.SlowThresholdMs = 3000
	}
	if m.ExpectedStatus == 0 {
		m.ExpectedStatus = 200
	}
	if m.AlertAfterFailures < 1 {
		m.AlertAfterFailures = 2
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	m.ConsecutiveFailures = 0
	m.LastStatus = models.StatusUnknown
	m.LastCheckedAt = nil
}

// InsertMonitorImport inserts a monitor preserving the exported id and created_at.
func (s *Store) InsertMonitorImport(m *models.Monitor) error {
	prepareMonitorImport(m)
	_, err := s.db.Exec(`
		INSERT INTO monitors (`+monitorColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, string(m.Type), m.Name, m.URL, m.Port, nullString(m.Config), m.Method,
		m.ExpectedStatus, m.ExpectedStatusMin, m.ExpectedStatusMax,
		nullString(m.KeywordMustExist), nullString(m.KeywordMustNotExist),
		nullString(m.RequestBody), nullString(m.RequestHeaders),
		m.HTTPUsername, m.HTTPPassword,
		m.IntervalSeconds, m.TimeoutMs, m.SlowThresholdMs,
		boolToInt(m.FollowRedirects), nullString(m.AlertEmails), boolToInt(m.Enabled),
		boolToInt(m.NotifyEmail), boolToInt(m.NotifySlack), boolToInt(m.NotifyWebhooks), boolToInt(m.Invert),
		encodeTags(m.Tags), nullString(m.HeartbeatToken), nullString(m.TenantID), m.AlertAfterFailures,
		m.ConsecutiveFailures, string(m.LastStatus), nil,
		formatTime(m.CreatedAt), formatTime(m.UpdatedAt),
	)
	return err
}

// ReplaceMonitorImport overwrites monitor configuration by id and resets runtime state.
func (s *Store) ReplaceMonitorImport(m *models.Monitor) error {
	prepareMonitorImport(m)
	res, err := s.db.Exec(`
		UPDATE monitors SET
			type=?, name=?, url=?, port=?, config=?, method=?, expected_status=?, expected_status_min=?, expected_status_max=?,
			keyword_must_exist=?, keyword_must_not_exist=?, request_body=?, request_headers=?, http_username=?, http_password=?,
			interval_seconds=?, timeout_ms=?, slow_threshold_ms=?, follow_redirects=?,
			alert_emails=?, enabled=?, notify_email=?, notify_slack=?, notify_webhooks=?, invert=?, tags=?, heartbeat_token=?, tenant_id=?, alert_after_failures=?,
			consecutive_failures=?, last_status=?, last_checked_at=?,
			updated_at=?
		WHERE id=?`,
		string(m.Type), m.Name, m.URL, m.Port, nullString(m.Config), m.Method,
		m.ExpectedStatus, m.ExpectedStatusMin, m.ExpectedStatusMax,
		nullString(m.KeywordMustExist), nullString(m.KeywordMustNotExist),
		nullString(m.RequestBody), nullString(m.RequestHeaders),
		m.HTTPUsername, m.HTTPPassword,
		m.IntervalSeconds, m.TimeoutMs, m.SlowThresholdMs, boolToInt(m.FollowRedirects),
		nullString(m.AlertEmails), boolToInt(m.Enabled),
		boolToInt(m.NotifyEmail), boolToInt(m.NotifySlack), boolToInt(m.NotifyWebhooks), boolToInt(m.Invert),
		encodeTags(m.Tags), nullString(m.HeartbeatToken), nullString(m.TenantID), m.AlertAfterFailures,
		m.ConsecutiveFailures, string(m.LastStatus), nil,
		formatTime(m.UpdatedAt), m.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("monitor not found")
	}
	return nil
}
