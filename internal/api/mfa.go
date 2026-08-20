package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

const (
	mfaCodeLength      = 8
	mfaCodeTTL         = 10 * time.Minute
	mfaCodeMaxAttempts = 5
	mfaIssueLimit      = 5
	mfaIssueWindow     = time.Hour
	mfaVerifyLimit     = 10
	mfaVerifyWindow    = 15 * time.Minute
)

var (
	errMFAEmailRequired       = errors.New("mfa email required")
	errMFADeliveryUnavailable = errors.New("mfa delivery unavailable")
	errMFATooManyIssues       = errors.New("too many verification emails, try again later")
)

type loginResponse struct {
	OK          bool   `json:"ok"`
	MFARequired bool   `json:"mfa_required,omitempty"`
	ChallengeID string `json:"challenge_id,omitempty"`
	EmailHint   string `json:"email_hint,omitempty"`
	Message     string `json:"message,omitempty"`
}

type verifyMFALoginRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type resendMFALoginRequest struct {
	ChallengeID string `json:"challenge_id"`
}

func mfaIssueKey(username string) string {
	return "mfa-issue:" + strings.ToLower(strings.TrimSpace(username))
}

func mfaVerifyKey(challengeID, ip string) string {
	return "mfa-verify:" + strings.TrimSpace(challengeID) + ":" + strings.TrimSpace(ip)
}

func hashMFACode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToUpper(code))))
	return hex.EncodeToString(sum[:])
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local, domain := parts[0], parts[1]
	if len(local) <= 2 {
		local = local[:1] + strings.Repeat("*", max(0, len(local)-1))
	} else {
		local = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
	}
	domainParts := strings.Split(domain, ".")
	if len(domainParts) > 0 && len(domainParts[0]) > 1 {
		domainParts[0] = domainParts[0][:1] + strings.Repeat("*", len(domainParts[0])-1)
	}
	return local + "@" + strings.Join(domainParts, ".")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func generateMFACode(n int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if n <= 0 {
		return "", fmt.Errorf("invalid code length")
	}
	buf := make([]byte, n)
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(raw[i])%len(alphabet)]
	}
	return string(buf), nil
}

func (s *Server) beginMFALogin(w http.ResponseWriter, r *http.Request, user *models.User) error {
	if strings.TrimSpace(user.Email) == "" {
		return errMFAEmailRequired
	}
	if s.sendMFACode == nil {
		return errMFADeliveryUnavailable
	}
	if !s.limits.Allow(mfaIssueKey(user.Username), mfaIssueLimit, mfaIssueWindow) {
		return errMFATooManyIssues
	}
	challengeID, err := randomToken(24)
	if err != nil {
		return err
	}
	code, err := generateMFACode(mfaCodeLength)
	if err != nil {
		return err
	}
	if err := s.store.CreateMFALoginChallenge(user.ID, challengeID, hashMFACode(code), time.Now().UTC().Add(mfaCodeTTL), mfaCodeMaxAttempts); err != nil {
		return err
	}
	if err := s.sendMFACode(user.Email, user.Username, code); err != nil {
		_ = s.store.DeleteMFALoginChallenge(challengeID)
		return errMFADeliveryUnavailable
	}
	jsonOK(w, loginResponse{
		OK:          false,
		MFARequired: true,
		ChallengeID: challengeID,
		EmailHint:   maskEmail(user.Email),
		Message:     "Verification code sent to your email.",
	})
	return nil
}

func (s *Server) handleVerifyMFALogin(w http.ResponseWriter, r *http.Request) {
	var req verifyMFALoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	challengeID := strings.TrimSpace(req.ChallengeID)
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if challengeID == "" || code == "" {
		jsonError(w, http.StatusBadRequest, "challenge and code required")
		return
	}
	ip := clientIP(r)
	if !s.limits.Allow(mfaVerifyKey(challengeID, ip), mfaVerifyLimit, mfaVerifyWindow) {
		jsonError(w, http.StatusTooManyRequests, "too many verification attempts, sign in again")
		return
	}
	challenge, err := s.store.GetMFALoginChallenge(challengeID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if challenge == nil {
		jsonError(w, http.StatusBadRequest, "verification expired, sign in again")
		return
	}
	user, err := s.store.GetUserByID(challenge.UserID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if user == nil || !user.MFAEnabled {
		_ = s.store.DeleteMFALoginChallenge(challengeID)
		jsonError(w, http.StatusBadRequest, "verification expired, sign in again")
		return
	}
	if hashMFACode(code) != challenge.CodeHash {
		remaining := challenge.AttemptsRemaining - 1
		if remaining <= 0 {
			_ = s.store.DeleteMFALoginChallenge(challengeID)
			s.recordSecurityEvent("mfa verification exhausted", auditActor(user.Username), "mfa_verify_lockout", "auth",
				"username="+user.Username+" ip="+ip,
				"endpoint", "mfa-verify", "ip", ip, "username", user.Username)
			jsonError(w, http.StatusTooManyRequests, "too many invalid codes, sign in again")
			return
		}
		_ = s.store.UseMFALoginChallengeAttempt(challengeID, remaining)
		jsonError(w, http.StatusUnauthorized, fmt.Sprintf("invalid verification code (%d attempts left)", remaining))
		return
	}
	_ = s.store.DeleteMFALoginChallenge(challengeID)
	if err := s.completeLogin(w, r, user); err != nil {
		jsonError(w, http.StatusInternalServerError, "session error")
		return
	}
	jsonOK(w, loginResponse{OK: true})
}

func (s *Server) handleResendMFALogin(w http.ResponseWriter, r *http.Request) {
	var req resendMFALoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	oldID := strings.TrimSpace(req.ChallengeID)
	if oldID == "" {
		jsonError(w, http.StatusBadRequest, "challenge required")
		return
	}
	challenge, err := s.store.GetMFALoginChallenge(oldID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if challenge == nil {
		jsonError(w, http.StatusBadRequest, "verification expired, sign in again")
		return
	}
	user, err := s.store.GetUserByID(challenge.UserID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if user == nil || !user.MFAEnabled || strings.TrimSpace(user.Email) == "" {
		_ = s.store.DeleteMFALoginChallenge(oldID)
		jsonError(w, http.StatusBadRequest, "verification expired, sign in again")
		return
	}
	if !s.limits.Allow(mfaIssueKey(user.Username), mfaIssueLimit, mfaIssueWindow) {
		jsonError(w, http.StatusTooManyRequests, "too many verification emails, try again later")
		return
	}
	newID, err := randomToken(24)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	code, err := generateMFACode(mfaCodeLength)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if err := s.store.CreateMFALoginChallenge(user.ID, newID, hashMFACode(code), time.Now().UTC().Add(mfaCodeTTL), mfaCodeMaxAttempts); err != nil {
		jsonInternal(w, err)
		return
	}
	if err := s.sendMFACode(user.Email, user.Username, code); err != nil {
		_ = s.store.DeleteMFALoginChallenge(newID)
		jsonError(w, http.StatusBadRequest, "mfa delivery unavailable")
		return
	}
	jsonOK(w, loginResponse{
		OK:          false,
		MFARequired: true,
		ChallengeID: newID,
		EmailHint:   maskEmail(user.Email),
		Message:     "A new verification code was sent.",
	})
}
