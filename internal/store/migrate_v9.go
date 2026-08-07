package store

func (s *Store) migrateV9() error {
	if err := s.addColumnIfMissing("monitors", "alert_after_failures",
		"ALTER TABLE monitors ADD COLUMN alert_after_failures INTEGER NOT NULL DEFAULT 2"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("performance_targets", "alert_after_slow",
		"ALTER TABLE performance_targets ADD COLUMN alert_after_slow INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("performance_targets", "consecutive_slow",
		"ALTER TABLE performance_targets ADD COLUMN consecutive_slow INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return nil
}
