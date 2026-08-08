package store

// migrateV12 removes legacy recovery incident rows. Recoveries are tracked by
// resolving the DOWN incident (resolved_at), not as separate always-open rows.
func (s *Store) migrateV12() error {
	_, err := s.db.Exec(`DELETE FROM incidents WHERE type = 'recovery'`)
	return err
}
