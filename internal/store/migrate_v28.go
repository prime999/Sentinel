package store

import "time"

// migrateV28 removes host log-collection schema left from the abandoned feature.
func (s *Store) migrateV28() error {
	if _, err := s.db.Exec(`
		DROP TABLE IF EXISTS log_alert_rules;
		DROP TABLE IF EXISTS log_sources;
		DELETE FROM settings WHERE key = 'logs';
	`); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE incidents SET resolved_at = ? WHERE type = 'host_log' AND resolved_at IS NULL`,
		formatTime(time.Now().UTC()),
	)
	return err
}
