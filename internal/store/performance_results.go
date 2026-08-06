package store

import (
	"database/sql"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) InsertPerformanceResult(r *models.PerformanceResult) error {
	r.ID = newID()
	_, err := s.db.Exec(`
		INSERT INTO performance_results
		(id, target_id, status, status_code, response_time_ms, dns_ms, tcp_ms, tls_ms, ttfb_ms, error, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.TargetID, string(r.Status), r.StatusCode, r.ResponseTimeMs,
		r.DNSMs, r.TCPMs, r.TLSMs, r.TTFBMs, r.Error, formatTime(r.CheckedAt),
	)
	return err
}

func (s *Store) ListPerformanceResults(targetID string, limit, offset int) ([]models.PerformanceResult, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, target_id, status, status_code, response_time_ms, dns_ms, tcp_ms, tls_ms, ttfb_ms, error, checked_at
		FROM performance_results WHERE target_id = ? ORDER BY checked_at DESC LIMIT ? OFFSET ?`,
		targetID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.PerformanceResult
	for rows.Next() {
		r, err := scanPerformanceResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *r)
	}
	return results, rows.Err()
}

func (s *Store) PruneOldPerformanceResults(before time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM performance_results WHERE checked_at < ?`, formatTime(before))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanPerformanceResult(row interface {
	Scan(dest ...any) error
}) (*models.PerformanceResult, error) {
	var r models.PerformanceResult
	var status string
	var statusCode, dnsMs, tcpMs, tlsMs, ttfbMs sql.NullInt64
	var checkedAt string

	err := row.Scan(
		&r.ID, &r.TargetID, &status, &statusCode,
		&r.ResponseTimeMs, &dnsMs, &tcpMs, &tlsMs, &ttfbMs,
		&r.Error, &checkedAt,
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
	if t, err := parseTime(checkedAt); err == nil {
		r.CheckedAt = t
	}
	return &r, nil
}
