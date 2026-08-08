package store

import (
	"database/sql"
	"sort"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func computePerformance(times []int, slowCount int, total int) models.PerformanceMetrics {
	if len(times) == 0 {
		return models.PerformanceMetrics{}
	}

	sorted := append([]int(nil), times...)
	sort.Ints(sorted)

	totalRT := 0
	for _, t := range times {
		totalRT += t
	}

	m := models.PerformanceMetrics{
		AvgMs:     totalRT / len(times),
		MinMs:     sorted[0],
		MaxMs:     sorted[len(sorted)-1],
		P50Ms:     percentile(sorted, 0.50),
		P95Ms:     percentile(sorted, 0.95),
		P99Ms:     percentile(sorted, 0.99),
		SlowCount: slowCount,
	}
	if total > 0 {
		m.DegradedPct = float64(slowCount) / float64(total) * 100
	}
	return m
}

func targetHealth(hasData bool, p95, threshold, slowCount, checkCount int) string {
	if !hasData {
		return "collecting"
	}
	slowPct := 0.0
	if checkCount > 0 {
		slowPct = float64(slowCount) / float64(checkCount) * 100
	}
	if slowPct > 0 || (threshold > 0 && p95 >= threshold) {
		return "warning"
	}
	return "good"
}

func fleetBucketDuration(since time.Time) time.Duration {
	span := time.Since(since)
	switch {
	case span > 14*24*time.Hour:
		return 24 * time.Hour
	case span > 48*time.Hour:
		return 6 * time.Hour
	default:
		return time.Hour
	}
}

func (s *Store) GetMonitorStats(monitorID string, since time.Time) (*models.MonitorStats, error) {
	rows, err := s.db.Query(`
		SELECT checked_at, response_time_ms, status, dns_ms, tcp_ms, tls_ms, ttfb_ms
		FROM check_results
		WHERE monitor_id = ? AND checked_at >= ?
		ORDER BY checked_at ASC`,
		monitorID, formatTime(since),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &models.MonitorStats{MonitorID: monitorID}
	var times []int
	var totalRT, upCount, slowCount, total int

	for rows.Next() {
		var checkedAt string
		var rt int
		var status string
		var dnsMs, tcpMs, tlsMs, ttfbMs sql.NullInt64
		if err := rows.Scan(&checkedAt, &rt, &status, &dnsMs, &tcpMs, &tlsMs, &ttfbMs); err != nil {
			return nil, err
		}

		t, _ := parseTime(checkedAt)
		stats.Points = append(stats.Points, models.StatsPoint{
			Timestamp:      t,
			ResponseTimeMs: rt,
			Status:         status,
			DNSMs:          nullableInt(dnsMs),
			TCPMs:          nullableInt(tcpMs),
			TLSMs:          nullableInt(tlsMs),
			TTFBMs:         nullableInt(ttfbMs),
		})
		times = append(times, rt)
		totalRT += rt
		total++
		if status == string(models.StatusDegraded) {
			slowCount++
		}
		if status == string(models.StatusUp) || status == string(models.StatusDegraded) {
			upCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if total > 0 {
		stats.AvgResponse = totalRT / total
		stats.UptimePct = float64(upCount) / float64(total) * 100
	}
	stats.Performance = computePerformance(times, slowCount, total)
	return stats, nil
}

func (s *Store) GetPerformanceTargetStats(targetID string, since time.Time) (*models.PerformanceStats, error) {
	rows, err := s.db.Query(`
		SELECT checked_at, response_time_ms, status, dns_ms, tcp_ms, tls_ms, ttfb_ms
		FROM performance_results
		WHERE target_id = ? AND checked_at >= ?
		ORDER BY checked_at ASC`,
		targetID, formatTime(since),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &models.PerformanceStats{TargetID: targetID}
	var times []int
	var totalRT, slowCount, total int

	for rows.Next() {
		var checkedAt string
		var rt int
		var status string
		var dnsMs, tcpMs, tlsMs, ttfbMs sql.NullInt64
		if err := rows.Scan(&checkedAt, &rt, &status, &dnsMs, &tcpMs, &tlsMs, &ttfbMs); err != nil {
			return nil, err
		}

		t, _ := parseTime(checkedAt)
		stats.Points = append(stats.Points, models.StatsPoint{
			Timestamp:      t,
			ResponseTimeMs: rt,
			Status:         status,
			DNSMs:          nullableInt(dnsMs),
			TCPMs:          nullableInt(tcpMs),
			TLSMs:          nullableInt(tlsMs),
			TTFBMs:         nullableInt(ttfbMs),
		})
		times = append(times, rt)
		totalRT += rt
		total++
		if status == string(models.StatusDegraded) || status == string(models.StatusDown) {
			slowCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if total > 0 {
		stats.AvgResponse = totalRT / total
	}
	stats.Performance = computePerformance(times, slowCount, total)
	return stats, nil
}

func (s *Store) GetPerformanceSlowStats(targetID string, since time.Time) (slowPct float64, total, slow int, err error) {
	var slowSum sql.NullInt64
	err = s.db.QueryRow(`
		SELECT COUNT(*),
			SUM(CASE WHEN status IN (?, ?) THEN 1 ELSE 0 END)
		FROM performance_results
		WHERE target_id = ? AND checked_at >= ?`,
		string(models.StatusDegraded), string(models.StatusDown),
		targetID, formatTime(since),
	).Scan(&total, &slowSum)
	if err != nil {
		return 0, 0, 0, err
	}
	if slowSum.Valid {
		slow = int(slowSum.Int64)
	}
	if total == 0 {
		return 0, 0, 0, nil
	}
	slowPct = float64(slow) / float64(total) * 100
	return slowPct, total, slow, nil
}

func (s *Store) GetFleetPerformance(since time.Time) (*models.FleetPerformance, error) {
	return s.GetFleetPerformanceScoped(since, "")
}

func (s *Store) GetFleetPerformanceByTenant(since time.Time, tenantID string) (*models.FleetPerformance, error) {
	if tenantID == "" {
		return &models.FleetPerformance{Monitors: []models.MonitorPerformance{}, Timeline: []models.FleetTimelinePoint{}}, nil
	}
	return s.GetFleetPerformanceScoped(since, tenantID)
}

func (s *Store) GetFleetPerformanceScoped(since time.Time, tenantID string) (*models.FleetPerformance, error) {
	type targetAgg struct {
		meta  models.MonitorPerformance
		times []int
		slow  int
	}

	targetsByID := map[string]*targetAgg{}
	q := `
		SELECT id, name, url, last_status, slow_threshold_ms
		FROM performance_targets WHERE enabled = 1`
	var args []any
	if tenantID != "" {
		q += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	q += ` ORDER BY created_at DESC`
	targetRows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer targetRows.Close()

	for targetRows.Next() {
		var id, name, url, lastStatus string
		var slowThreshold int
		if err := targetRows.Scan(&id, &name, &url, &lastStatus, &slowThreshold); err != nil {
			return nil, err
		}
		if lastStatus == string(models.StatusDown) {
			lastStatus = string(models.StatusDegraded)
		}
		targetsByID[id] = &targetAgg{meta: models.MonitorPerformance{
			MonitorID: id, MonitorName: name, Type: "http", URL: url,
			Status: lastStatus, SlowThresholdMs: slowThreshold,
		}}
	}
	if err := targetRows.Err(); err != nil {
		return nil, err
	}

	bucketDur := fleetBucketDuration(since)
	type bucketAgg struct {
		totalRT   int
		count     int
		slowCount int
		start     time.Time
	}
	buckets := map[int64]*bucketAgg{}
	var fleetTimes []int
	var totalChecks, totalSlow, totalRT int

	resultRows, err := s.db.Query(`
		SELECT target_id, checked_at, response_time_ms, status
		FROM performance_results
		WHERE checked_at >= ?
		ORDER BY checked_at ASC`, formatTime(since))
	if err != nil {
		return nil, err
	}
	defer resultRows.Close()

	for resultRows.Next() {
		var targetID, checkedAt, status string
		var rt int
		if err := resultRows.Scan(&targetID, &checkedAt, &rt, &status); err != nil {
			return nil, err
		}
		t, err := parseTime(checkedAt)
		if err != nil {
			continue
		}

		a, ok := targetsByID[targetID]
		if !ok {
			continue
		}
		a.times = append(a.times, rt)
		a.meta.CheckCount++
		if status == string(models.StatusDegraded) {
			a.slow++
		}
		if status == string(models.StatusDown) {
			a.slow++
		}

		fleetTimes = append(fleetTimes, rt)
		totalChecks++
		totalRT += rt
		if status == string(models.StatusDegraded) || status == string(models.StatusDown) {
			totalSlow++
		}

		bucketStart := t.Truncate(bucketDur)
		key := bucketStart.Unix()
		b, ok := buckets[key]
		if !ok {
			b = &bucketAgg{start: bucketStart}
			buckets[key] = b
		}
		b.totalRT += rt
		b.count++
		if status == string(models.StatusDegraded) || status == string(models.StatusDown) {
			b.slowCount++
		}
	}
	if err := resultRows.Err(); err != nil {
		return nil, err
	}

	fleet := &models.FleetPerformance{
		MonitorCount: len(targetsByID),
		TotalChecks:  totalChecks,
		SlowChecks:   totalSlow,
	}
	if totalChecks > 0 {
		fleet.AvgMs = totalRT / totalChecks
	}
	fleetPerf := computePerformance(fleetTimes, totalSlow, totalChecks)
	fleet.P95Ms = fleetPerf.P95Ms

	timeline := make([]models.FleetTimelinePoint, 0, len(buckets))
	for _, b := range buckets {
		avg := 0
		if b.count > 0 {
			avg = b.totalRT / b.count
		}
		timeline = append(timeline, models.FleetTimelinePoint{
			Timestamp:  b.start,
			AvgMs:      avg,
			CheckCount: b.count,
			SlowCount:  b.slowCount,
		})
	}
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})
	fleet.Timeline = timeline

	services := make([]models.MonitorPerformance, 0, len(targetsByID))
	for _, a := range targetsByID {
		hasData := len(a.times) > 0
		if hasData {
			perf := computePerformance(a.times, a.slow, a.meta.CheckCount)
			a.meta.AvgMs = perf.AvgMs
			a.meta.P95Ms = perf.P95Ms
			a.meta.MaxMs = perf.MaxMs
			a.meta.SlowCount = a.slow
		}
		a.meta.HasData = hasData
		a.meta.Health = targetHealth(hasData, a.meta.P95Ms, a.meta.SlowThresholdMs, a.meta.SlowCount, a.meta.CheckCount)

		switch a.meta.Health {
		case "good":
			fleet.HealthyCount++
		case "warning":
			fleet.WarningCount++
		case "collecting":
			fleet.CollectingCount++
		}
		services = append(services, a.meta)
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].HasData != services[j].HasData {
			return services[i].HasData
		}
		return services[i].P95Ms > services[j].P95Ms
	})
	fleet.Monitors = services
	return fleet, nil
}
