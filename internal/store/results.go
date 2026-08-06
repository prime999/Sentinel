package store

import (
	"database/sql"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) InsertCheckResult(r *models.CheckResult) error {
	r.ID = newID()
	_, err := s.db.Exec(`
		INSERT INTO check_results
		(id, monitor_id, status, status_code, response_time_ms, dns_ms, tcp_ms, tls_ms, ttfb_ms, error, details, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.MonitorID, string(r.Status), r.StatusCode, r.ResponseTimeMs,
		r.DNSMs, r.TCPMs, r.TLSMs, r.TTFBMs, r.Error, r.Details, formatTime(r.CheckedAt),
	)
	return err
}

func (s *Store) ListCheckResults(monitorID string, limit, offset int) ([]models.CheckResult, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, monitor_id, status, status_code, response_time_ms, dns_ms, tcp_ms, tls_ms, ttfb_ms, error, details, checked_at
		FROM check_results WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT ? OFFSET ?`,
		monitorID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.CheckResult
	for rows.Next() {
		r, err := scanCheckResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *r)
	}
	return results, rows.Err()
}

func scanCheckResult(row interface {
	Scan(dest ...any) error
}) (*models.CheckResult, error) {
	var r models.CheckResult
	var status string
	var statusCode, dnsMs, tcpMs, tlsMs, ttfbMs sql.NullInt64
	var details sql.NullString
	var checkedAt string

	err := row.Scan(
		&r.ID, &r.MonitorID, &status, &statusCode,
		&r.ResponseTimeMs, &dnsMs, &tcpMs, &tlsMs, &ttfbMs,
		&r.Error, &details, &checkedAt,
	)
	if err != nil {
		return nil, err
	}

	r.Status = models.MonitorStatus(status)
	if statusCode.Valid {
		v := int(statusCode.Int64)
		r.StatusCode = &v
	}
	r.DNSMs = nullableInt(dnsMs)
	r.TCPMs = nullableInt(tcpMs)
	r.TLSMs = nullableInt(tlsMs)
	r.TTFBMs = nullableInt(ttfbMs)
	if details.Valid {
		r.Details = details.String
	}
	if t, err := parseTime(checkedAt); err == nil {
		r.CheckedAt = t
	}
	return &r, nil
}

func (s *Store) PruneOldResults(before time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM check_results WHERE checked_at < ?`, formatTime(before))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
