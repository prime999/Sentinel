package store

func (s *Store) migrateV3() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`)
	if err != nil {
		return err
	}
	return s.addColumnIfMissing("sessions", "user_id", "ALTER TABLE sessions ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE CASCADE")
}
