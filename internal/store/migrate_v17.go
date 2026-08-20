package store

func (s *Store) migrateV17() error {
	if err := s.addColumnIfMissing("users", "mfa_enabled", "ALTER TABLE users ADD COLUMN mfa_enabled INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS login_mfa_challenges (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			code_hash TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			attempts_remaining INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`)
	return err
}
