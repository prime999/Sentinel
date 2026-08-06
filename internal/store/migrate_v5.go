package store

func (s *Store) migrateV5() error {
	if err := s.addColumnIfMissing("users", "email", "ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`)
	return err
}
