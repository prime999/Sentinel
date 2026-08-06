package store

func (s *Store) migrateV8() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS customers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			monitor_quota INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		)`)
	if err != nil {
		return err
	}

	if err := s.addColumnIfMissing("users", "tenant_id", "ALTER TABLE users ADD COLUMN tenant_id TEXT REFERENCES customers(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("monitors", "tenant_id", "ALTER TABLE monitors ADD COLUMN tenant_id TEXT REFERENCES customers(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("performance_targets", "tenant_id", "ALTER TABLE performance_targets ADD COLUMN tenant_id TEXT REFERENCES customers(id) ON DELETE SET NULL"); err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_monitors_tenant_id ON monitors(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_performance_targets_tenant_id ON performance_targets(tenant_id)`)
	return err
}
