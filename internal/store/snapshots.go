package store

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) GetSnapshot(monitorID, snapshotType string) (string, error) {
	var data string
	err := s.db.QueryRow(`
		SELECT data FROM monitor_snapshots WHERE monitor_id = ? AND snapshot_type = ?`,
		monitorID, snapshotType,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return data, err
}

func (s *Store) SaveSnapshot(monitorID, snapshotType, data string) error {
	now := formatTime(time.Now().UTC())
	_, err := s.db.Exec(`
		INSERT INTO monitor_snapshots (monitor_id, snapshot_type, data, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(monitor_id, snapshot_type) DO UPDATE SET
			data = excluded.data, updated_at = excluded.updated_at`,
		monitorID, snapshotType, data, now,
	)
	return err
}

func (s *Store) GetLastAlertedSSLDays(monitorID string) (int, error) {
	raw, err := s.GetSnapshot(monitorID, "ssl_alert")
	if err != nil || raw == "" {
		return 0, nil
	}
	var days int
	if _, err := fmt.Sscanf(raw, "%d", &days); err != nil {
		return 0, nil
	}
	return days, nil
}

func (s *Store) SaveLastAlertedSSLDays(monitorID string, days int) error {
	return s.SaveSnapshot(monitorID, "ssl_alert", fmt.Sprintf("%d", days))
}
