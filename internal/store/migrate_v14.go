package store

// migrateV14 closes leftover open "slow" incidents on uptime monitors.
// Latency alerts belong to performance targets; stale slow rows skewed "Last Incident".
func (s *Store) migrateV14() error {
	_, err := s.db.Exec(`
		UPDATE incidents
		SET resolved_at = COALESCE(resolved_at, started_at)
		WHERE type = 'slow'
		  AND resolved_at IS NULL
		  AND monitor_id IN (SELECT id FROM monitors)
	`)
	return err
}
