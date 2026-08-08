package store

// migrateV11 clears latency-based "degraded" on uptime monitors.
// Uptime is up/down only; SSL may still use degraded for cert warnings.
// Latency warnings belong on performance targets.
func (s *Store) migrateV11() error {
	_, err := s.db.Exec(`
		UPDATE monitors
		SET last_status = 'up'
		WHERE last_status = 'degraded' AND type != 'ssl'
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		UPDATE incidents
		SET resolved_at = datetime('now')
		WHERE type = 'slow' AND resolved_at IS NULL
		  AND monitor_id IN (SELECT id FROM monitors WHERE type != 'ssl')
	`)
	return err
}
