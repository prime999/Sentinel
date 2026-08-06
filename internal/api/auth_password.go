package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		jsonError(w, http.StatusBadRequest, "email required")
		return
	}

	user, err := s.store.GetUserByEmail(email)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	const sentMsg = "Password reset link sent to your email."
	if user == nil {
		jsonOK(w, map[string]string{"message": sentMsg})
		return
	}

	token, err := randomToken(32)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "token error")
		return
	}
	expires := time.Now().UTC().Add(time.Hour)
	if err := s.store.CreatePasswordResetToken(user.ID, token, expires); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resetURL := strings.TrimRight(s.dashboardURL, "/") + "/reset-password?token=" + token
	to, username := user.Email, user.Username
	go func() {
		if err := s.alerter.SendPasswordResetEmail(to, username, resetURL); err != nil {
			log.Printf("password reset email: %v", err)
		}
	}()

	jsonOK(w, map[string]string{"message": sentMsg})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		jsonError(w, http.StatusBadRequest, "token and new password required")
		return
	}
	if len(req.NewPassword) < 6 {
		jsonError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	userID, err := s.store.GetPasswordResetUserID(req.Token)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if userID == "" {
		jsonError(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}

	user, err := s.store.GetUserByID(userID)
	if err != nil || user == nil {
		jsonError(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}

	updated, err := s.store.UpdateUser(userID, user.Username, user.Email, user.Role, req.NewPassword, user.TenantID, false)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.store.DeletePasswordResetToken(req.Token)
	to, username := updated.Email, updated.Username
	go func() {
		if err := s.alerter.SendPasswordChangedEmail(to, username); err != nil {
			log.Printf("password changed email: %v", err)
		}
	}()

	jsonOK(w, map[string]bool{"ok": true})
}
