package store

func (s *Store) migrateV4() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS performance_targets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'GET',
			interval_seconds INTEGER NOT NULL DEFAULT 300,
			timeout_ms INTEGER NOT NULL DEFAULT 10000,
			slow_threshold_ms INTEGER NOT NULL DEFAULT 3000,
			follow_redirects INTEGER NOT NULL DEFAULT 1,
			enabled INTEGER NOT NULL DEFAULT 1,
			last_status TEXT NOT NULL DEFAULT 'unknown',
			last_checked_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS performance_results (
			id TEXT PRIMARY KEY,
			target_id TEXT NOT NULL REFERENCES performance_targets(id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			status_code INTEGER,
			response_time_ms INTEGER NOT NULL DEFAULT 0,
			dns_ms INTEGER,
			tcp_ms INTEGER,
			tls_ms INTEGER,
			ttfb_ms INTEGER,
			error TEXT,
			checked_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_performance_results_target_checked
			ON performance_results(target_id, checked_at DESC)`)
	return err
}
