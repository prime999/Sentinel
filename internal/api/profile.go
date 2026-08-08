package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/sentinel-monitoring/sentinel/internal/store"
)

type profileResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id,omitempty"`
}

type updateProfileRequest struct {
	CurrentPassword string `json:"current_password"`
	Username        string `json:"username"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jsonOK(w, profileResponse{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
		Role:     string(user.Role),
		TenantID: user.TenantID,
	})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.CurrentPassword == "" {
		jsonError(w, http.StatusBadRequest, "current password required")
		return
	}

	fresh, err := s.store.GetUserByID(user.ID)
	if err != nil || fresh == nil {
		jsonError(w, http.StatusInternalServerError, "user not found")
		return
	}
	if !store.CheckPassword(fresh.PasswordHash, req.CurrentPassword) {
		jsonError(w, http.StatusUnauthorized, "incorrect current password")
		return
	}

	username := fresh.Username
	if strings.TrimSpace(req.Username) != "" {
		username = strings.TrimSpace(req.Username)
	}
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)

	updated, err := s.store.UpdateOwnProfile(fresh.ID, username, name, email, req.NewPassword)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.NewPassword != "" {
		to, uname := updated.Email, updated.Username
		go func() {
			if err := s.alerter.SendPasswordChangedEmail(to, uname); err != nil {
				log.Printf("password changed email: %v", err)
			}
		}()
	}

	jsonOK(w, profileResponse{
		ID:       updated.ID,
		Username: updated.Username,
		Name:     updated.Name,
		Email:    updated.Email,
		Role:     string(updated.Role),
		TenantID: updated.TenantID,
	})
}
