package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/safehost"
)

const (
	monitorBackupFormat  = "sentinel-monitors"
	monitorBackupVersion = 1
)

type monitorBackupFile struct {
	Format        string           `json:"format"`
	Version       int              `json:"version"`
	ExportedAt    time.Time        `json:"exported_at"`
	ScopeTenantID string           `json:"scope_tenant_id,omitempty"`
	Monitors      []models.Monitor `json:"monitors"`
}

type monitorBackupImportBody struct {
	Format   string           `json:"format,omitempty"`
	Version  int              `json:"version,omitempty"`
	Monitors []models.Monitor `json:"monitors"`
	Mode     string           `json:"mode"`
}

type monitorBackupConflict struct {
	ID           string `json:"id"`
	ImportName   string `json:"import_name"`
	ExistingName string `json:"existing_name"`
}

type monitorBackupInvalid struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error"`
}

type monitorBackupPreview struct {
	Total        int                     `json:"total"`
	ToCreate     int                     `json:"to_create"`
	Conflicts    []monitorBackupConflict `json:"conflicts"`
	Invalid      []monitorBackupInvalid  `json:"invalid"`
	QuotaBlocked int                     `json:"quota_blocked"`
	WouldUpdate  int                     `json:"would_update"`
	WouldSkip    int                     `json:"would_skip"`
}

type monitorBackupImportResult struct {
	Created int                    `json:"created"`
	Updated int                    `json:"updated"`
	Skipped int                    `json:"skipped"`
	Failed  []monitorBackupInvalid `json:"failed"`
}

func monitorForExport(m models.Monitor) models.Monitor {
	m.ConsecutiveFailures = 0
	m.LastStatus = models.StatusUnknown
	m.LastCheckedAt = nil
	m.HTTPAuthSet = strings.TrimSpace(m.HTTPUsername) != "" || m.HTTPPassword != ""
	return m
}

func (s *Server) handleExportMonitorBackup(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	tenantScope := exportTenantScope(r, user)

	file, err := s.buildPlatformExport(user, tenantScope)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		jsonInternal(w, err)
		return
	}
	filename := fmt.Sprintf("sentinel-platform-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	detail := fmt.Sprintf("customers=%d users=%d monitors=%d", len(file.Customers), len(file.Users), len(file.Monitors))
	_ = s.store.InsertAudit(user.Username, "export", "platform_backup", detail)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handlePreviewMonitorBackup(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	file, _, err := parsePlatformBackupBody(body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(file.Monitors) == 0 && len(file.Customers) == 0 && file.Format != platformBackupFormat {
		jsonError(w, http.StatusBadRequest, "no data in backup")
		return
	}
	createPreview, err := s.previewPlatformBackup(user, file, "create_only")
	if err != nil {
		jsonInternal(w, err)
		return
	}
	overwritePreview, err := s.previewPlatformBackup(user, file, "overwrite")
	if err != nil {
		jsonInternal(w, err)
		return
	}
	mergePreviewWouldUpdate(createPreview, overwritePreview)
	jsonOK(w, createPreview)
}

func mergePreviewWouldUpdate(createPrev, overwritePrev *platformBackupPreview) {
	createPrev.Customers.WouldUpdate = overwritePrev.Customers.WouldUpdate
	createPrev.Users.WouldUpdate = overwritePrev.Users.WouldUpdate
	createPrev.Monitors.WouldUpdate = overwritePrev.Monitors.WouldUpdate
	createPrev.PerformanceTargets.WouldUpdate = overwritePrev.PerformanceTargets.WouldUpdate
	createPrev.Hosts.WouldUpdate = overwritePrev.Hosts.WouldUpdate
	createPrev.MaintenanceWindows.WouldUpdate = overwritePrev.MaintenanceWindows.WouldUpdate
}

func (s *Server) handleImportMonitorBackup(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	file, mode, err := parsePlatformBackupBody(body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if mode != "create_only" && mode != "overwrite" {
		jsonError(w, http.StatusBadRequest, "mode must be create_only or overwrite")
		return
	}
	if len(file.Monitors) == 0 && len(file.Customers) == 0 && len(file.Users) == 0 {
		jsonError(w, http.StatusBadRequest, "no data in backup")
		return
	}

	result := normalizePlatformImportResult(s.importPlatformBackup(user, file, mode))
	detail := fmt.Sprintf("monitors c=%d u=%d s=%d f=%d", result.Monitors.Created, result.Monitors.Updated, result.Monitors.Skipped, len(result.Monitors.Failed))
	_ = s.store.InsertAudit(user.Username, "import", "platform_backup", detail)
	jsonOK(w, result)
}

func exportTenantScope(r *http.Request, user *models.User) string {
	if isCustomerAdmin(user) {
		return user.TenantID
	}
	if !isPlatformAdmin(user) {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("customer"))
}

func monitorAccessibleByUser(user *models.User, m models.Monitor) bool {
	if isPlatformAdmin(user) {
		return true
	}
	if isCustomerAdmin(user) {
		return m.TenantID == user.TenantID
	}
	return false
}

func (s *Server) normalizeBackupMonitor(user *models.User, m *models.Monitor) error {
	if isCustomerAdmin(user) {
		m.TenantID = user.TenantID
		return nil
	}
	if !isPlatformAdmin(user) {
		return fmt.Errorf("forbidden")
	}
	m.TenantID = strings.TrimSpace(m.TenantID)
	return nil
}

func validateBackupMonitor(m *models.Monitor) error {
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("monitor id is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if m.IntervalSeconds != 0 && m.IntervalSeconds < 30 {
		return fmt.Errorf("interval must be at least 30 seconds")
	}
	return safehost.ValidateMonitorTarget(string(m.Type), m.URL, m.Port)
}

func (s *Server) buildMonitorBackupPreview(user *models.User, monitors []models.Monitor, mode string) (*monitorBackupPreview, error) {
	preview := &monitorBackupPreview{
		Total:     len(monitors),
		Conflicts: []monitorBackupConflict{},
		Invalid:   []monitorBackupInvalid{},
	}
	ids := make([]string, 0, len(monitors))
	for _, m := range monitors {
		if strings.TrimSpace(m.ID) != "" {
			ids = append(ids, m.ID)
		}
	}
	existing, err := s.store.GetMonitorsByIDs(ids)
	if err != nil {
		return nil, err
	}

	tenantNewCounts := map[string]int{}

	for _, raw := range monitors {
		m := raw
		if err := s.normalizeBackupMonitor(user, &m); err != nil {
			preview.Invalid = append(preview.Invalid, monitorBackupInvalid{ID: m.ID, Name: m.Name, Error: err.Error()})
			continue
		}
		if err := validateBackupMonitor(&m); err != nil {
			preview.Invalid = append(preview.Invalid, monitorBackupInvalid{ID: m.ID, Name: m.Name, Error: err.Error()})
			continue
		}

		ex, ok := existing[m.ID]
		if ok {
			if !monitorAccessibleByUser(user, ex) {
				preview.WouldSkip++
				continue
			}
			preview.Conflicts = append(preview.Conflicts, monitorBackupConflict{
				ID:           m.ID,
				ImportName:   m.Name,
				ExistingName: ex.Name,
			})
			if mode == "overwrite" {
				preview.WouldUpdate++
			} else {
				preview.WouldSkip++
			}
			continue
		}

		preview.ToCreate++
		if m.TenantID != "" {
			tenantNewCounts[m.TenantID]++
		}
	}

	for tenantID, n := range tenantNewCounts {
		c, err := s.store.GetCustomer(tenantID)
		if err != nil || c == nil {
			continue
		}
		count, err := s.store.CountMonitorsByTenant(tenantID)
		if err != nil {
			return nil, err
		}
		remaining := c.MonitorQuota - count
		if remaining < n {
			blocked := n - remaining
			if blocked < 0 {
				blocked = 0
			}
			preview.QuotaBlocked += blocked
		}
	}

	return preview, nil
}
