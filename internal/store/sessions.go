package store

import (
	"database/sql"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) CreateSession(id, userID string, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`, id, userID, formatTime(expiresAt))
	return err
}

func (s *Store) GetSessionUser(id string) (*models.User, error) {
	var expiresAt, userID string
	err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE id = ?`, id).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := parseTime(expiresAt)
	if err != nil || time.Now().UTC().After(t) {
		s.DeleteSession(id)
		return nil, nil
	}
	if userID == "" {
		return nil, nil
	}
	return s.GetUserByID(userID)
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) PruneSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, formatTime(time.Now().UTC()))
	return err
}
