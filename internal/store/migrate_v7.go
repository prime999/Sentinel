package store

func (s *Store) migrateV7() error {
	if err := s.addColumnIfMissing("monitors", "tags", "ALTER TABLE monitors ADD COLUMN tags TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("monitors", "heartbeat_token", "ALTER TABLE monitors ADD COLUMN heartbeat_token TEXT"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("performance_targets", "alert_emails", "ALTER TABLE performance_targets ADD COLUMN alert_emails TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err := s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_monitors_heartbeat_token ON monitors(heartbeat_token);
		CREATE TABLE IF NOT EXISTS maintenance_windows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			monitor_id TEXT,
			starts_at TEXT NOT NULL,
			ends_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			detail TEXT,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
		CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			prefix TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT
		)`)
	return err
}
