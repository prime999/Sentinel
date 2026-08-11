package store

func (s *Store) migrateV15() error {
	if err := s.addColumnIfMissing("monitors", "notify_email",
		"ALTER TABLE monitors ADD COLUMN notify_email INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("monitors", "notify_slack",
		"ALTER TABLE monitors ADD COLUMN notify_slack INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	return s.addColumnIfMissing("monitors", "notify_webhooks",
		"ALTER TABLE monitors ADD COLUMN notify_webhooks INTEGER NOT NULL DEFAULT 1")
}
