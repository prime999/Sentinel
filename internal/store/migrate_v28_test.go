package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateV28DropsLogCollection(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "sentinel.db")
	sidecar := filepath.Join(dir, "sentinel-logs.db")
	if err := os.WriteFile(sidecar, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}

	// Seed via raw SQL before Open so migrateV28 can clean it.
	db, err := sql.Open("sqlite3", mainPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE incidents (
			id TEXT PRIMARY KEY, monitor_id TEXT, type TEXT, message TEXT,
			started_at TEXT, resolved_at TEXT
		);
		CREATE TABLE hosts (id TEXT PRIMARY KEY);
		CREATE TABLE log_sources (id TEXT PRIMARY KEY, host_id TEXT);
		CREATE TABLE log_alert_rules (id TEXT PRIMARY KEY, host_id TEXT);
		INSERT INTO settings(key, value) VALUES ('logs', '{}');
		INSERT INTO incidents(id, monitor_id, type, message, started_at)
			VALUES ('i1', 'h1', 'host_log', 'x', '2020-01-01T00:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	st, err := Open(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var n int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('log_sources','log_alert_rules')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected log tables dropped, found %d", n)
	}
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key='logs'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("expected logs settings key removed")
	}
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE type='host_log' AND resolved_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("expected host_log incidents resolved")
	}
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Fatalf("expected sidecar removed, err=%v", err)
	}
}
