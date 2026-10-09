package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/monitorhost"
)

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	customerFilter := strings.TrimSpace(r.URL.Query().Get("customer"))
	var sites []models.Site
	var err error
	if isPlatformAdmin(user) {
		if customerFilter != "" {
			sites, err = s.store.ListSites(customerFilter)
		} else {
			sites, err = s.store.ListSites("")
		}
	} else if user.TenantID != "" {
		sites, err = s.store.ListSites(user.TenantID)
	} else {
		sites = []models.Site{}
	}
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if sites == nil {
		sites = []models.Site{}
	}
	jsonOK(w, sites)
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	var site models.Site
	if err := json.Unmarshal(body, &site); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	site.Name = strings.TrimSpace(site.Name)
	if site.Name == "" {
		jsonError(w, http.StatusBadRequest, "name is required")
		return
	}
	if isCustomerAdmin(user) {
		site.TenantID = user.TenantID
	} else if !isPlatformAdmin(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	} else {
		site.TenantID = strings.TrimSpace(site.TenantID)
	}
	site.PrimaryHost = monitorhost.NormalizeHost(site.PrimaryHost)
	if site.PrimaryHost == "" && strings.TrimSpace(site.Name) != "" {
		site.PrimaryHost = monitorhost.NormalizeHost(site.Name)
	}
	if err := s.store.CreateSite(&site); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "create", "site", site.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, site)
}

func (s *Server) handleUpdateSite(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	existing, err := s.store.GetSite(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || !tenantResourceAccessible(user, existing.TenantID) {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if !canWriteResources(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	var input models.Site
	if err := json.Unmarshal(body, &input); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if n := strings.TrimSpace(input.Name); n != "" {
		existing.Name = n
	}
	if h := strings.TrimSpace(input.PrimaryHost); h != "" {
		existing.PrimaryHost = monitorhost.NormalizeHost(h)
	}
	if err := s.store.UpdateSite(existing); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "update", "site", existing.Name)
	jsonOK(w, existing)
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	existing, err := s.store.GetSite(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || !tenantResourceAccessible(user, existing.TenantID) {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if !canWriteResources(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.store.DeleteSite(id); err != nil {
		if strings.Contains(err.Error(), "has monitors") {
			jsonError(w, http.StatusConflict, "site has monitors")
			return
		}
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "delete", "site", existing.Name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) validateMonitorSite(user *models.User, siteID, tenantID string) error {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return nil
	}
	site, err := s.store.GetSite(siteID)
	if err != nil {
		return err
	}
	if site == nil || !canAccessTenant(user, site.TenantID) {
		return errSiteNotFound
	}
	if strings.TrimSpace(site.TenantID) != strings.TrimSpace(tenantID) {
		return errSiteTenantMismatch
	}
	return nil
}

var (
	errSiteNotFound       = &siteValidationError{msg: "site not found"}
	errSiteTenantMismatch = &siteValidationError{msg: "site tenant mismatch"}
)

type siteValidationError struct{ msg string }

func (e *siteValidationError) Error() string { return e.msg }
