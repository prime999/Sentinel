package store

import (
	"database/sql"
	"fmt"
)

func (s *Store) migrateV2() error {
	cols := []struct{ table, column, ddl string }{
		{"monitors", "type", "ALTER TABLE monitors ADD COLUMN type TEXT NOT NULL DEFAULT 'http'"},
		{"monitors", "port", "ALTER TABLE monitors ADD COLUMN port INTEGER"},
		{"monitors", "config", "ALTER TABLE monitors ADD COLUMN config TEXT"},
		{"check_results", "details", "ALTER TABLE check_results ADD COLUMN details TEXT"},
	}
	for _, c := range cols {
		if err := s.addColumnIfMissing(c.table, c.column, c.ddl); err != nil {
			return err
		}
	}

	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS monitor_snapshots (
			monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
			snapshot_type TEXT NOT NULL,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (monitor_id, snapshot_type)
		)`)
	return err
}

func (s *Store) addColumnIfMissing(table, column, ddl string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = s.db.Exec(ddl)
	return err
}
