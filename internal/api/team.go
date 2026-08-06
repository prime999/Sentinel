package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

type teamMemberResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TenantID  string `json:"tenant_id,omitempty"`
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

func toTeamMemberResponse(u models.User) teamMemberResponse {
	return teamMemberResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      string(u.Role),
		TenantID:  u.TenantID,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (s *Server) handleListTeam(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var users []models.User
	var err error
	if isPlatformAdmin(user) {
		users, err = s.store.ListUsers()
	} else if isCustomerAdmin(user) {
		users, err = s.store.ListUsersByTenant(user.TenantID)
	} else {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]teamMemberResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toTeamMemberResponse(u))
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
		if tenantID == "" && role == models.RoleViewer {
			// Allow platform-scoped viewers historically, but prefer customer assignment.
		}
	} else {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}

	user, err := s.store.CreateUser(strings.TrimSpace(req.Username), strings.TrimSpace(req.Email), req.Password, role, tenantID)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, toTeamMemberResponse(*user))
}

func (s *Server) handleUpdateTeamMember(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	var req updateTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	existing, err := s.store.GetUserByID(id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}

	if isCustomerAdmin(actor) {
		if existing.TenantID != actor.TenantID {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
	} else if !isPlatformAdmin(actor) {
		jsonError(w, http.StatusForbidden, "forbidden")
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
			jsonError(w, http.StatusInternalServerError, err.Error())
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
	jsonOK(w, toTeamMemberResponse(*user))
}

func (s *Server) handleDeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	current := currentUser(r)
	if current != nil && current.ID == id {
		jsonError(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	target, err := s.store.GetUserByID(id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if target == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}

	if isCustomerAdmin(current) {
		if target.TenantID != current.TenantID {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
	} else if !isPlatformAdmin(current) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}

	if target.Role == models.RoleAdmin && target.TenantID == "" {
		admins, err := s.store.CountPlatformAdmins()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
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
	w.WriteHeader(http.StatusNoContent)
}
