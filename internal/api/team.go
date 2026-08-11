package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

type teamMemberResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TenantID  string `json:"tenant_id,omitempty"`
	Locked    bool   `json:"locked"`
	CreatedAt string `json:"created_at"`
}

type createTeamMemberRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
}

type updateTeamMemberRequest struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
	TenantID *string `json:"tenant_id"`
}

type resetTeamPasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) toTeamMemberResponse(u models.User) teamMemberResponse {
	locked := false
	if s.limits != nil {
		locked = s.limits.Count(loginFailKey(u.Username), loginFailWindow) >= loginFailLimit
	}
	return teamMemberResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      string(u.Role),
		TenantID:  u.TenantID,
		Locked:    locked,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// loadManagedUser returns the target user if the actor may manage them.
func (s *Server) loadManagedUser(actor *models.User, id string) (*models.User, int, string) {
	existing, err := s.store.GetUserByID(id)
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	if existing == nil {
		return nil, http.StatusNotFound, "not found"
	}
	if isCustomerAdmin(actor) {
		if existing.TenantID != actor.TenantID {
			return nil, http.StatusNotFound, "not found"
		}
	} else if !isPlatformAdmin(actor) {
		return nil, http.StatusForbidden, "forbidden"
	}
	return existing, 0, ""
}

func (s *Server) clearLoginLockout(username string) {
	if s.limits == nil {
		return
	}
	key := loginFailKey(username)
	s.limits.Clear(key)
	s.limits.Clear("lockout-notice:" + key)
}

func (s *Server) handleListTeam(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	var users []models.User
	var err error
	if isPlatformAdmin(actor) {
		users, err = s.store.ListUsers()
	} else if isCustomerAdmin(actor) {
		users, err = s.store.ListUsersByTenant(actor.TenantID)
	} else {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		jsonInternal(w, err)
		return
	}
	out := make([]teamMemberResponse, 0, len(users))
	for _, u := range users {
		out = append(out, s.toTeamMemberResponse(u))
	}
	jsonOK(w, out)
}

func (s *Server) handleCreateTeamMember(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	var req createTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	role := models.UserRole(strings.TrimSpace(req.Role))
	if role == "" {
		role = models.RoleViewer
	}
	if role != models.RoleAdmin && role != models.RoleViewer {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	tenantID := strings.TrimSpace(req.TenantID)
	if isCustomerAdmin(actor) {
		tenantID = actor.TenantID
	} else if isPlatformAdmin(actor) {
		// Platform admins: empty tenant = platform user; set tenant for customer users.
	} else {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}

	user, err := s.store.CreateUser(strings.TrimSpace(req.Username), strings.TrimSpace(req.Email), req.Password, role, tenantID)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.store.InsertAudit(actor.Username, "create", "user", user.Username)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, s.toTeamMemberResponse(*user))
}

func (s *Server) handleUpdateTeamMember(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	var req updateTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	existing, code, msg := s.loadManagedUser(actor, id)
	if existing == nil {
		if code == http.StatusInternalServerError {
			jsonInternal(w, fmt.Errorf("%s", msg))
			return
		}
		jsonError(w, code, msg)
		return
	}

	role := existing.Role
	if req.Role != "" {
		role = models.UserRole(strings.TrimSpace(req.Role))
	}
	if role != models.RoleAdmin && role != models.RoleViewer {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	wasPlatformAdmin := existing.Role == models.RoleAdmin && existing.TenantID == ""
	newTenantID := existing.TenantID
	if isCustomerAdmin(actor) {
		newTenantID = actor.TenantID
	} else if req.TenantID != nil {
		newTenantID = strings.TrimSpace(*req.TenantID)
	}
	willBePlatformAdmin := role == models.RoleAdmin && newTenantID == ""
	if wasPlatformAdmin && !willBePlatformAdmin {
		admins, err := s.store.CountPlatformAdmins()
		if err != nil {
			jsonInternal(w, err)
			return
		}
		if admins <= 1 {
			jsonError(w, http.StatusBadRequest, "cannot remove the last platform admin")
			return
		}
	}

	username := existing.Username
	if req.Username != "" {
		username = strings.TrimSpace(req.Username)
	}
	email := existing.Email
	if req.Email != "" {
		email = strings.TrimSpace(req.Email)
	}

	tenantID := existing.TenantID
	updateTenant := false
	if isCustomerAdmin(actor) {
		tenantID = actor.TenantID
		updateTenant = true
	} else if req.TenantID != nil {
		tenantID = strings.TrimSpace(*req.TenantID)
		updateTenant = true
	}

	user, err := s.store.UpdateUser(id, username, email, role, req.Password, tenantID, updateTenant)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Password != "" {
		s.clearLoginLockout(user.Username)
	}
	_ = s.store.InsertAudit(actor.Username, "update", "user", user.Username)
	jsonOK(w, s.toTeamMemberResponse(*user))
}

func (s *Server) handleUnlockTeamMember(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	existing, code, msg := s.loadManagedUser(actor, id)
	if existing == nil {
		if code == http.StatusInternalServerError {
			jsonInternal(w, fmt.Errorf("%s", msg))
			return
		}
		jsonError(w, code, msg)
		return
	}
	s.clearLoginLockout(existing.Username)
	ip := clientIP(r)
	_ = s.store.InsertAudit(actor.Username, "unlock", "auth",
		"username="+existing.Username+" ip="+ip)
	jsonOK(w, s.toTeamMemberResponse(*existing))
}

func (s *Server) handleResetTeamMemberPassword(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	var req resetTeamPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		jsonError(w, http.StatusBadRequest, "password required")
		return
	}
	if err := store.ValidatePassword(req.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, code, msg := s.loadManagedUser(actor, id)
	if existing == nil {
		if code == http.StatusInternalServerError {
			jsonInternal(w, fmt.Errorf("%s", msg))
			return
		}
		jsonError(w, code, msg)
		return
	}

	user, err := s.store.UpdateUser(id, existing.Username, existing.Email, existing.Role, req.Password, existing.TenantID, false)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.clearLoginLockout(user.Username)
	_ = s.store.InsertAudit(actor.Username, "password_reset", "user", user.Username)
	jsonOK(w, s.toTeamMemberResponse(*user))
}

func (s *Server) handleDeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	current := currentUser(r)
	if current != nil && current.ID == id {
		jsonError(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	target, code, msg := s.loadManagedUser(current, id)
	if target == nil {
		if code == http.StatusInternalServerError {
			jsonInternal(w, fmt.Errorf("%s", msg))
			return
		}
		jsonError(w, code, msg)
		return
	}

	if target.Role == models.RoleAdmin && target.TenantID == "" {
		admins, err := s.store.CountPlatformAdmins()
		if err != nil {
			jsonInternal(w, err)
			return
		}
		if admins <= 1 {
			jsonError(w, http.StatusBadRequest, "cannot delete the last platform admin")
			return
		}
	}

	if err := s.store.DeleteUser(id); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.store.InsertAudit(current.Username, "delete", "user", target.Username)
	w.WriteHeader(http.StatusNoContent)
}
