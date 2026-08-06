package store

import (
	"database/sql"
	"time"
)

func (s *Store) CreatePasswordResetToken(userID, token string, expiresAt time.Time) error {
	_, err := s.db.Exec(`DELETE FROM password_reset_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO password_reset_tokens (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)`,
		token, userID, formatTime(expiresAt), formatTime(time.Now().UTC()),
	)
	return err
}

func (s *Store) GetPasswordResetUserID(token string) (string, error) {
	var userID, expiresAt string
	err := s.db.QueryRow(`
		SELECT user_id, expires_at FROM password_reset_tokens WHERE token = ?`, token,
	).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	t, err := parseTime(expiresAt)
	if err != nil || time.Now().UTC().After(t) {
		s.DeletePasswordResetToken(token)
		return "", nil
	}
	return userID, nil
}

func (s *Store) DeletePasswordResetToken(token string) error {
	_, err := s.db.Exec(`DELETE FROM password_reset_tokens WHERE token = ?`, token)
	return err
}

func (s *Store) PruneExpiredResetTokens() error {
	_, err := s.db.Exec(`DELETE FROM password_reset_tokens WHERE expires_at < ?`, formatTime(time.Now().UTC()))
	return err
}
