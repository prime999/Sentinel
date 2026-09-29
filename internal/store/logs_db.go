package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

// LogsDB holds high-churn log events separate from the main Sentinel DB.
type LogsDB struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

func logsDBPath(mainPath string) string {
	dir := filepath.Dir(mainPath)
	base := filepath.Base(mainPath)
	if strings.HasSuffix(base, ".db") {
		return filepath.Join(dir, strings.TrimSuffix(base, ".db")+"-logs.db")
	}
	return filepath.Join(dir, "sentinel-logs.db")
}

func OpenLogsDB(mainPath string) (*LogsDB, error) {
	path := logsDBPath(mainPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
		db.Close()
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	l := &LogsDB{db: db, path: path}
	if err := l.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return l, nil
}

func (l *LogsDB) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

func (l *LogsDB) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *LogsDB) migrate() error {
	_, err := l.db.Exec(`
		CREATE TABLE IF NOT EXISTS log_events (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			ts TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			fingerprint TEXT NOT NULL DEFAULT '',
			metadata TEXT NOT NULL DEFAULT '',
			received_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_log_events_host_src_ts ON log_events(host_id, source_id, ts);
		CREATE INDEX IF NOT EXISTS idx_log_events_fp_ts ON log_events(fingerprint, ts);
		CREATE INDEX IF NOT EXISTS idx_log_events_ts ON log_events(ts);

		CREATE TABLE IF NOT EXISTS log_volume_buckets (
			host_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			bucket_start TEXT NOT NULL,
			level TEXT NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (host_id, source_id, bucket_start, level)
		);
		CREATE INDEX IF NOT EXISTS idx_log_volume_ts ON log_volume_buckets(bucket_start);
	`)
	return err
}

func (l *LogsDB) FileSize() (int64, error) {
	if l == nil {
		return 0, nil
	}
	info, err := os.Stat(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

func (l *LogsDB) InsertEventsBatch(events []models.LogEvent) error {
	if l == nil || len(events) == 0 {
		return nil
	}
	tx, err := l.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
		INSERT INTO log_events (id, host_id, source_id, ts, level, message, fingerprint, metadata, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		id := e.ID
		if id == "" {
			id = newID()
		}
		recv := e.ReceivedAt
		if recv.IsZero() {
			recv = time.Now().UTC()
		}
		if _, err := stmt.Exec(
			id, e.HostID, e.SourceID, formatTime(e.Timestamp), string(e.Level),
			e.Message, e.Fingerprint, e.Metadata, formatTime(recv),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (l *LogsDB) UpsertVolumeBatch(buckets []models.LogVolumeBucket) error {
	if l == nil || len(buckets) == 0 {
		return nil
	}
	tx, err := l.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
		INSERT INTO log_volume_buckets (host_id, source_id, bucket_start, level, count)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(host_id, source_id, bucket_start, level)
		DO UPDATE SET count = count + excluded.count`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, b := range buckets {
		if _, err := stmt.Exec(b.HostID, b.SourceID, formatTime(b.BucketStart), string(b.Level), b.Count); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type LogEventQuery struct {
	HostID   string
	SourceID string
	Level    string
	Search   string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

func (l *LogsDB) QueryEvents(q LogEventQuery) ([]models.LogEvent, error) {
	if l == nil {
		return nil, nil
	}
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 500 {
		q.Limit = 500
	}
	conds := []string{"1=1"}
	args := []any{}
	if q.HostID != "" {
		conds = append(conds, "host_id = ?")
		args = append(args, q.HostID)
	}
	if q.SourceID != "" {
		conds = append(conds, "source_id = ?")
		args = append(args, q.SourceID)
	}
	if q.Level != "" {
		conds = append(conds, "level = ?")
		args = append(args, q.Level)
	}
	if q.Search != "" {
		conds = append(conds, "message LIKE ?")
		args = append(args, "%"+q.Search+"%")
	}
	if q.From != nil {
		conds = append(conds, "ts >= ?")
		args = append(args, formatTime(*q.From))
	}
	if q.To != nil {
		conds = append(conds, "ts < ?")
		args = append(args, formatTime(*q.To))
	}
	args = append(args, q.Limit, q.Offset)
	rows, err := l.db.Query(`
		SELECT id, host_id, source_id, ts, level, message, fingerprint, metadata, received_at
		FROM log_events WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY ts DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogEvent
	for rows.Next() {
		var e models.LogEvent
		var ts, recv, level string
		if err := rows.Scan(&e.ID, &e.HostID, &e.SourceID, &ts, &level, &e.Message, &e.Fingerprint, &e.Metadata, &recv); err != nil {
			return nil, err
		}
		e.Level = models.LogLevel(level)
		if t, err := parseTime(ts); err == nil {
			e.Timestamp = t
		}
		if t, err := parseTime(recv); err == nil {
			e.ReceivedAt = t
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (l *LogsDB) QueryVolume(hostID, sourceID string, from, to time.Time) ([]models.LogVolumeBucket, error) {
	if l == nil {
		return nil, nil
	}
	rows, err := l.db.Query(`
		SELECT host_id, source_id, bucket_start, level, count
		FROM log_volume_buckets
		WHERE host_id = ? AND (? = '' OR source_id = ?)
		  AND bucket_start >= ? AND bucket_start < ?
		ORDER BY bucket_start ASC`,
		hostID, sourceID, sourceID, formatTime(from), formatTime(to),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogVolumeBucket
	for rows.Next() {
		var b models.LogVolumeBucket
		var start, level string
		if err := rows.Scan(&b.HostID, &b.SourceID, &start, &level, &b.Count); err != nil {
			return nil, err
		}
		b.Level = models.LogLevel(level)
		if t, err := parseTime(start); err == nil {
			b.BucketStart = t
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (l *LogsDB) CountEventsSince(hostID, sourceID, pattern string, since time.Time) (int, error) {
	if l == nil {
		return 0, nil
	}
	q := `SELECT COUNT(*) FROM log_events WHERE host_id = ? AND ts >= ?`
	args := []any{hostID, formatTime(since)}
	if sourceID != "" {
		q += ` AND source_id = ?`
		args = append(args, sourceID)
	}
	if pattern != "" {
		q += ` AND message LIKE ?`
		args = append(args, "%"+pattern+"%")
	}
	var n int
	err := l.db.QueryRow(q, args...).Scan(&n)
	return n, err
}

// PruneEvents deletes old events in batches. Returns total deleted.
func (l *LogsDB) PruneEvents(before time.Time, batchSize int) (int64, error) {
	if l == nil {
		return 0, nil
	}
	if batchSize < 1 {
		batchSize = 10000
	}
	var total int64
	cutoff := formatTime(before)
	for {
		res, err := l.db.Exec(`
			DELETE FROM log_events WHERE id IN (
				SELECT id FROM log_events WHERE ts < ? ORDER BY ts ASC LIMIT ?
			)`, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		n, _ := res.RowsAffected()
		total += n
		if n < int64(batchSize) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return total, nil
}

func (l *LogsDB) PruneVolume(before time.Time, batchSize int) (int64, error) {
	if l == nil {
		return 0, nil
	}
	if batchSize < 1 {
		batchSize = 10000
	}
	var total int64
	cutoff := formatTime(before)
	for {
		res, err := l.db.Exec(`
			DELETE FROM log_volume_buckets WHERE rowid IN (
				SELECT rowid FROM log_volume_buckets WHERE bucket_start < ? ORDER BY bucket_start ASC LIMIT ?
			)`, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		n, _ := res.RowsAffected()
		total += n
		if n < int64(batchSize) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return total, nil
}

func (l *LogsDB) EnsureUnderSizeCap(maxBytes int64, retentionDays int) error {
	if l == nil || maxBytes <= 0 {
		return nil
	}
	size, err := l.FileSize()
	if err != nil {
		return err
	}
	if size < maxBytes {
		return nil
	}
	// Aggressive prune: oldest half of retention window
	days := retentionDays
	if days < 1 {
		days = 1
	}
	before := time.Now().UTC().AddDate(0, 0, -days/2)
	if _, err := l.PruneEvents(before, 20000); err != nil {
		return err
	}
	if _, err := l.PruneVolume(before, 20000); err != nil {
		return err
	}
	size, err = l.FileSize()
	if err != nil {
		return err
	}
	if size >= maxBytes {
		return fmt.Errorf("log store full (%d bytes, cap %d)", size, maxBytes)
	}
	return nil
}
