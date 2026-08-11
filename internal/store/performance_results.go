package store

import (
	"database/sql"
	"strings"
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

type PerformanceResultQuery struct {
	TargetID    string
	ThresholdMs int
	From, To    *time.Time
	Limit       int
	Offset      int
	// BreachesOnly keeps rows where response_time_ms exceeds the SLA threshold.
	BreachesOnly bool
}

func (s *Store) ListPerformanceResults(targetID string, limit, offset int) ([]models.PerformanceResult, error) {
	items, _, err := s.QueryPerformanceResults(PerformanceResultQuery{
		TargetID: targetID,
		Limit:    limit,
		Offset:   offset,
	})
	return items, err
}

func (s *Store) QueryPerformanceResults(q PerformanceResultQuery) ([]models.PerformanceResult, int, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	where := []string{"target_id = ?"}
	args := []any{q.TargetID}
	if q.BreachesOnly {
		if q.ThresholdMs <= 0 {
			return []models.PerformanceResult{}, 0, nil
		}
		where = append(where, "response_time_ms > ?")
		args = append(args, q.ThresholdMs)
	}
	if q.From != nil {
		where = append(where, "checked_at >= ?")
		args = append(args, formatTime(*q.From))
	}
	if q.To != nil {
		where = append(where, "checked_at < ?")
		args = append(args, formatTime(*q.To))
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM performance_results WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), q.Limit, q.Offset)
	rows, err := s.db.Query(`
		SELECT id, target_id, status, status_code, response_time_ms, dns_ms, tcp_ms, tls_ms, ttfb_ms, error, checked_at
		FROM performance_results
		WHERE `+clause+`
		ORDER BY checked_at DESC
		LIMIT ? OFFSET ?`,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []models.PerformanceResult
	for rows.Next() {
		r, err := scanPerformanceResult(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, *r)
	}
	return results, total, rows.Err()
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
