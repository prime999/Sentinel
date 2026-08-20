package store

func (s *Store) migrateV16() error {
	if err := s.addColumnIfMissing("monitors", "http_username",
		"ALTER TABLE monitors ADD COLUMN http_username TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return s.addColumnIfMissing("monitors", "http_password",
		"ALTER TABLE monitors ADD COLUMN http_password TEXT NOT NULL DEFAULT ''")
}
