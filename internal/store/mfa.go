package store

import (
	"database/sql"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) CreateMFALoginChallenge(userID, challengeID, codeHash string, expiresAt time.Time, attempts int) error {
	if _, err := s.db.Exec(`DELETE FROM login_mfa_challenges WHERE user_id = ?`, userID); err != nil {
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO login_mfa_challenges (id, user_id, code_hash, expires_at, attempts_remaining, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		challengeID, userID, codeHash, formatTime(expiresAt), attempts, formatTime(time.Now().UTC()),
	)
	return err
}

func (s *Store) GetMFALoginChallenge(id string) (*models.LoginMFAChallenge, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, code_hash, expires_at, attempts_remaining, created_at
		FROM login_mfa_challenges
		WHERE id = ?`, id)
	var challenge models.LoginMFAChallenge
	var expiresAt, createdAt string
	err := row.Scan(&challenge.ID, &challenge.UserID, &challenge.CodeHash, &expiresAt, &challenge.AttemptsRemaining, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	challenge.ExpiresAt, err = parseTime(expiresAt)
	if err != nil {
		return nil, err
	}
	challenge.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(challenge.ExpiresAt) {
		_ = s.DeleteMFALoginChallenge(id)
		return nil, nil
	}
	return &challenge, nil
}

func (s *Store) UpdateMFALoginChallenge(id, codeHash string, expiresAt time.Time, attempts int) error {
	_, err := s.db.Exec(`
		UPDATE login_mfa_challenges
		SET code_hash = ?, expires_at = ?, attempts_remaining = ?, created_at = ?
		WHERE id = ?`,
		codeHash, formatTime(expiresAt), attempts, formatTime(time.Now().UTC()), id,
	)
	return err
}

func (s *Store) UseMFALoginChallengeAttempt(id string, attemptsRemaining int) error {
	_, err := s.db.Exec(`UPDATE login_mfa_challenges SET attempts_remaining = ? WHERE id = ?`, attemptsRemaining, id)
	return err
}

func (s *Store) DeleteMFALoginChallenge(id string) error {
	_, err := s.db.Exec(`DELETE FROM login_mfa_challenges WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteMFALoginChallengesForUser(userID string) error {
	_, err := s.db.Exec(`DELETE FROM login_mfa_challenges WHERE user_id = ?`, userID)
	return err
}
