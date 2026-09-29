package store

func (s *Store) migrateV28() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS log_sources (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'file',
			path TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			enabled INTEGER NOT NULL DEFAULT 1,
			min_level TEXT NOT NULL DEFAULT 'WARN',
			include_patterns TEXT NOT NULL DEFAULT '[]',
			health TEXT NOT NULL DEFAULT 'ok',
			health_detail TEXT NOT NULL DEFAULT '',
			dropped_lines INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_log_sources_host ON log_sources(host_id);

		CREATE TABLE IF NOT EXISTS log_alert_rules (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
			source_id TEXT REFERENCES log_sources(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			pattern TEXT NOT NULL,
			threshold INTEGER NOT NULL DEFAULT 5,
			window_seconds INTEGER NOT NULL DEFAULT 300,
			severity TEXT NOT NULL DEFAULT 'warning',
			enabled INTEGER NOT NULL DEFAULT 1,
			notify_email INTEGER NOT NULL DEFAULT 1,
			notify_slack INTEGER NOT NULL DEFAULT 1,
			notify_webhooks INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_log_alert_rules_host ON log_alert_rules(host_id);
	`)
	return err
}
