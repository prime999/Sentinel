package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/alerter"
	"github.com/sentinel-monitoring/sentinel/internal/config"
	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

func boolPtr(v bool) *bool { return &v }

func newTestMFAServer(t *testing.T) (*Server, *store.Store, *models.User) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sentinel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Username: "admin",
			Password: "Admin1234",
		},
		Server: config.ServerConfig{
			DashboardURL: "http://localhost:8082",
		},
		SMTP: models.SMTPConfig{
			Host:    "smtp.example.com",
			Port:    587,
			From:    "alerts@example.com",
			Enabled: true,
		},
	}
	a := alerter.New(st, cfg.SMTP, cfg.SMTP, cfg.Server.DashboardURL)
	srv := New(st, a, cfg)
	user, err := st.GetUserByUsername("admin")
	if err != nil || user == nil {
		t.Fatalf("default admin: %v user=%v", err, user)
	}
	return srv, st, user
}

func postJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeLoginResponse(t *testing.T, rec *httptest.ResponseRecorder) loginResponse {
	t.Helper()
	var out loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	return out
}

func TestLoginWithoutMFAStillCreatesSession(t *testing.T) {
	srv, _, _ := newTestMFAServer(t)

	rec := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": "admin",
		"password": "Admin1234",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeLoginResponse(t, rec)
	if !resp.OK || resp.MFARequired {
		t.Fatalf("unexpected login response: %+v", resp)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "sentinel_session=") {
		t.Fatalf("expected session cookie, got %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestMFALoginRequiresVerificationAndCreatesSessionAfterCode(t *testing.T) {
	srv, st, user := newTestMFAServer(t)
	updated, err := st.UpdateOwnProfile(user.ID, user.Username, user.Name, "admin@example.com", "", boolPtr(true))
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}

	var sentCode string
	srv.sendMFACode = func(to, username, code string) error {
		if to != updated.Email || username != updated.Username {
			t.Fatalf("unexpected delivery target to=%q username=%q", to, username)
		}
		sentCode = code
		return nil
	}

	loginRec := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": updated.Username,
		"password": "Admin1234",
	})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}
	loginResp := decodeLoginResponse(t, loginRec)
	if !loginResp.MFARequired || loginResp.ChallengeID == "" {
		t.Fatalf("expected mfa challenge, got %+v", loginResp)
	}
	if sentCode == "" {
		t.Fatal("expected MFA code to be sent")
	}
	if cookie := loginRec.Header().Get("Set-Cookie"); cookie != "" {
		t.Fatalf("did not expect session cookie before verification, got %q", cookie)
	}

	verifyRec := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
		"challenge_id": loginResp.ChallengeID,
		"code":         sentCode,
	})
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", verifyRec.Code, verifyRec.Body.String())
	}
	verifyResp := decodeLoginResponse(t, verifyRec)
	if !verifyResp.OK {
		t.Fatalf("expected verified login, got %+v", verifyResp)
	}
	if !strings.Contains(verifyRec.Header().Get("Set-Cookie"), "sentinel_session=") {
		t.Fatalf("expected session cookie, got %q", verifyRec.Header().Get("Set-Cookie"))
	}
}

func TestMFALoginResendReplacesOldChallenge(t *testing.T) {
	srv, st, user := newTestMFAServer(t)
	updated, err := st.UpdateOwnProfile(user.ID, user.Username, user.Name, "admin@example.com", "", boolPtr(true))
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}

	var sentCodes []string
	srv.sendMFACode = func(to, username, code string) error {
		sentCodes = append(sentCodes, code)
		return nil
	}

	loginRec := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": updated.Username,
		"password": "Admin1234",
	})
	loginResp := decodeLoginResponse(t, loginRec)

	resendRec := postJSON(t, srv.Handler(), "/api/auth/mfa/resend", map[string]string{
		"challenge_id": loginResp.ChallengeID,
	})
	if resendRec.Code != http.StatusOK {
		t.Fatalf("resend status=%d body=%s", resendRec.Code, resendRec.Body.String())
	}
	resendResp := decodeLoginResponse(t, resendRec)
	if resendResp.ChallengeID == "" || resendResp.ChallengeID == loginResp.ChallengeID {
		t.Fatalf("expected replacement challenge, got %+v", resendResp)
	}
	if len(sentCodes) != 2 {
		t.Fatalf("expected 2 delivered codes, got %d", len(sentCodes))
	}

	oldVerify := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
		"challenge_id": loginResp.ChallengeID,
		"code":         sentCodes[0],
	})
	if oldVerify.Code != http.StatusBadRequest {
		t.Fatalf("expected expired old challenge, got status=%d body=%s", oldVerify.Code, oldVerify.Body.String())
	}

	newVerify := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
		"challenge_id": resendResp.ChallengeID,
		"code":         sentCodes[1],
	})
	if newVerify.Code != http.StatusOK {
		t.Fatalf("expected new code to work, status=%d body=%s", newVerify.Code, newVerify.Body.String())
	}
}

func TestMFALoginRejectsExpiredAndRepeatedWrongCodes(t *testing.T) {
	srv, st, user := newTestMFAServer(t)
	updated, err := st.UpdateOwnProfile(user.ID, user.Username, user.Name, "admin@example.com", "", boolPtr(true))
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}

	var sentCode string
	srv.sendMFACode = func(to, username, code string) error {
		sentCode = code
		return nil
	}

	loginRec := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": updated.Username,
		"password": "Admin1234",
	})
	loginResp := decodeLoginResponse(t, loginRec)
	if sentCode == "" {
		t.Fatal("missing sent code")
	}

	for i := 0; i < mfaCodeMaxAttempts-1; i++ {
		rec := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
			"challenge_id": loginResp.ChallengeID,
			"code":         "WRONG999",
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong attempt %d status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
	lastWrong := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
		"challenge_id": loginResp.ChallengeID,
		"code":         "WRONG999",
	})
	if lastWrong.Code != http.StatusTooManyRequests {
		t.Fatalf("expected lockout after wrong attempts, status=%d body=%s", lastWrong.Code, lastWrong.Body.String())
	}

	loginRec2 := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": updated.Username,
		"password": "Admin1234",
	})
	loginResp2 := decodeLoginResponse(t, loginRec2)
	_, err = st.DB().Exec(`UPDATE login_mfa_challenges SET expires_at = ? WHERE id = ?`, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano), loginResp2.ChallengeID)
	if err != nil {
		t.Fatalf("expire challenge: %v", err)
	}
	expiredRec := postJSON(t, srv.Handler(), "/api/auth/mfa/verify", map[string]string{
		"challenge_id": loginResp2.ChallengeID,
		"code":         sentCode,
	})
	if expiredRec.Code != http.StatusBadRequest {
		t.Fatalf("expected expired code rejection, status=%d body=%s", expiredRec.Code, expiredRec.Body.String())
	}
}

func TestMFALoginFailsWhenDeliveryUnavailable(t *testing.T) {
	srv, st, user := newTestMFAServer(t)
	updated, err := st.UpdateOwnProfile(user.ID, user.Username, user.Name, "admin@example.com", "", boolPtr(true))
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}
	srv.sendMFACode = func(to, username, code string) error {
		return fmt.Errorf("smtp down")
	}

	loginRec := postJSON(t, srv.Handler(), "/api/auth/login", map[string]string{
		"username": updated.Username,
		"password": "Admin1234",
	})
	if loginRec.Code != http.StatusBadRequest {
		t.Fatalf("expected delivery failure, status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(loginRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body["error"] != "mfa delivery unavailable" {
		t.Fatalf("unexpected error body: %+v", body)
	}
	rows, err := st.DB().Query(`SELECT id FROM login_mfa_challenges`)
	if err != nil {
		t.Fatalf("query challenges: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("challenge should be deleted when delivery fails")
	}
}
