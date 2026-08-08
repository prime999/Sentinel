package store

// migrateV13 closes informational SSL/DNS change incidents that were left open.
// Cert renewals and DNS changes are notices, not ongoing outages.
func (s *Store) migrateV13() error {
	_, err := s.db.Exec(`
		UPDATE incidents
		SET resolved_at = COALESCE(resolved_at, started_at)
		WHERE type IN ('cert_change', 'dns_change')
		  AND resolved_at IS NULL
	`)
	return err
}
