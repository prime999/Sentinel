package store

func (s *Store) migrateV10() error {
	return s.addColumnIfMissing("users", "name",
		"ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT ''")
}
