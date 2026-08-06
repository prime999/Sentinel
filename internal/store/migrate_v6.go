package store

func (s *Store) migrateV6() error {
	return s.addColumnIfMissing("monitors", "invert", "ALTER TABLE monitors ADD COLUMN invert INTEGER NOT NULL DEFAULT 0")
}
