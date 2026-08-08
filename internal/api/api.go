package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/alerter"
	"github.com/sentinel-monitoring/sentinel/internal/config"
	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/safehost"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

type Server struct {
	store        *store.Store
	alerter      *alerter.Alerter
	defaultSMTP  models.SMTPConfig
	dashboardURL string
	mux          *http.ServeMux
}

func New(s *store.Store, a *alerter.Alerter, cfg *config.Config) *Server {
	if err := s.EnsureDefaultAdmin(cfg.Auth); err != nil {
		panic("ensure default admin: " + err.Error())
	}
	srv := &Server{
		store:        s,
		alerter:      a,
		defaultSMTP:  cfg.SMTP,
		dashboardURL: cfg.Server.DashboardURL,
		mux:          http.NewServeMux(),
	}
	srv.routes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return s.cors(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("POST /api/auth/forgot-password", s.handleForgotPassword)
	s.mux.HandleFunc("POST /api/auth/reset-password", s.handleResetPassword)

	s.mux.HandleFunc("GET /api/monitors", s.authRequired(s.handleListMonitors))
	s.mux.HandleFunc("POST /api/monitors", s.adminRequired(s.handleCreateMonitor))
	s.mux.HandleFunc("GET /api/monitors/{id}", s.authRequired(s.handleGetMonitor))
	s.mux.HandleFunc("PUT /api/monitors/{id}", s.adminRequired(s.handleUpdateMonitor))
	s.mux.HandleFunc("DELETE /api/monitors/{id}", s.adminRequired(s.handleDeleteMonitor))
	s.mux.HandleFunc("GET /api/monitors/{id}/results", s.authRequired(s.handleListResults))
	s.mux.HandleFunc("GET /api/monitors/{id}/incidents", s.authRequired(s.handleListMonitorIncidents))
	s.mux.HandleFunc("GET /api/monitors/{id}/stats", s.authRequired(s.handleGetStats))
	s.mux.HandleFunc("GET /api/performance", s.authRequired(s.handleGetFleetPerformance))
	s.mux.HandleFunc("GET /api/performance/targets", s.authRequired(s.handleListPerformanceTargets))
	s.mux.HandleFunc("POST /api/performance/targets", s.adminRequired(s.handleCreatePerformanceTarget))
	s.mux.HandleFunc("GET /api/performance/targets/{id}", s.authRequired(s.handleGetPerformanceTarget))
	s.mux.HandleFunc("PUT /api/performance/targets/{id}", s.adminRequired(s.handleUpdatePerformanceTarget))
	s.mux.HandleFunc("DELETE /api/performance/targets/{id}", s.adminRequired(s.handleDeletePerformanceTarget))
	s.mux.HandleFunc("GET /api/performance/targets/{id}/results", s.authRequired(s.handleListPerformanceResults))
	s.mux.HandleFunc("GET /api/performance/targets/{id}/stats", s.authRequired(s.handleGetPerformanceStats))

	s.mux.HandleFunc("GET /api/settings/general", s.platformAdminRequired(s.handleGetGeneral))
	s.mux.HandleFunc("PUT /api/settings/general", s.platformAdminRequired(s.handlePutGeneral))
	s.mux.HandleFunc("POST /api/settings/general/reset", s.platformAdminRequired(s.handleResetGeneral))

	s.mux.HandleFunc("GET /api/settings/smtp", s.platformAdminRequired(s.handleGetSMTP))
	s.mux.HandleFunc("PUT /api/settings/smtp", s.platformAdminRequired(s.handlePutSMTP))
	s.mux.HandleFunc("POST /api/settings/smtp/test", s.platformAdminRequired(s.handleTestSMTP))

	s.mux.HandleFunc("GET /api/settings/team", s.adminRequired(s.handleListTeam))
	s.mux.HandleFunc("POST /api/settings/team", s.adminRequired(s.handleCreateTeamMember))
	s.mux.HandleFunc("PUT /api/settings/team/{id}", s.adminRequired(s.handleUpdateTeamMember))
	s.mux.HandleFunc("DELETE /api/settings/team/{id}", s.adminRequired(s.handleDeleteTeamMember))

	s.mux.HandleFunc("GET /api/settings/customers", s.platformAdminRequired(s.handleListCustomers))
	s.mux.HandleFunc("POST /api/settings/customers", s.platformAdminRequired(s.handleCreateCustomer))
	s.mux.HandleFunc("PUT /api/settings/customers/{id}", s.platformAdminRequired(s.handleUpdateCustomer))
	s.mux.HandleFunc("DELETE /api/settings/customers/{id}", s.platformAdminRequired(s.handleDeleteCustomer))

	s.mux.HandleFunc("GET /api/profile", s.authRequired(s.handleGetProfile))
	s.mux.HandleFunc("PUT /api/profile", s.authRequired(s.handleUpdateProfile))

	s.mux.HandleFunc("GET /api/incidents", s.authRequired(s.handleListIncidents))

	s.mux.HandleFunc("GET /api/settings/webhooks", s.platformAdminRequired(s.handleGetWebhooks))
	s.mux.HandleFunc("PUT /api/settings/webhooks", s.platformAdminRequired(s.handlePutWebhooks))
	s.mux.HandleFunc("GET /api/settings/maintenance", s.platformAdminRequired(s.handleListMaintenance))
	s.mux.HandleFunc("POST /api/settings/maintenance", s.platformAdminRequired(s.handleCreateMaintenance))
	s.mux.HandleFunc("DELETE /api/settings/maintenance/{id}", s.platformAdminRequired(s.handleDeleteMaintenance))
	s.mux.HandleFunc("GET /api/settings/server", s.platformAdminRequired(s.handleGetServerSettings))
	s.mux.HandleFunc("PUT /api/settings/server", s.platformAdminRequired(s.handlePutServerSettings))
	s.mux.HandleFunc("GET /api/settings/status-page", s.platformAdminRequired(s.handleGetStatusPageConfig))
	s.mux.HandleFunc("PUT /api/settings/status-page", s.platformAdminRequired(s.handlePutStatusPageConfig))
	s.mux.HandleFunc("GET /api/settings/audit", s.platformAdminRequired(s.handleListAudit))
	s.mux.HandleFunc("GET /api/settings/tokens", s.authRequired(s.handleListAPITokens))
	s.mux.HandleFunc("POST /api/settings/tokens", s.authRequired(s.handleCreateAPIToken))
	s.mux.HandleFunc("DELETE /api/settings/tokens/{id}", s.authRequired(s.handleDeleteAPIToken))

	s.mux.HandleFunc("GET /api/public/status", s.handlePublicStatus)
	s.mux.HandleFunc("POST /api/heartbeat/{token}", s.handleHeartbeatPing)
	s.mux.HandleFunc("GET /api/heartbeat/{token}", s.handleHeartbeatPing)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	user, err := s.store.GetUserByUsername(req.Username)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if user == nil || !store.CheckPassword(user.PasswordHash, req.Password) {
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	sessionID, err := randomToken(32)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "session error")
		return
	}
	expires := time.Now().UTC().Add(24 * time.Hour)
	if err := s.store.CreateSession(sessionID, user.ID, expires); err != nil {
		jsonError(w, http.StatusInternalServerError, "session error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sentinel_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(r, s.dashboardURL),
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
	jsonOK(w, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("sentinel_session"); err == nil {
		_ = s.store.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sentinel_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(r, s.dashboardURL),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	jsonOK(w, map[string]bool{"ok": true})
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	tag := r.URL.Query().Get("tag")
	customerFilter := strings.TrimSpace(r.URL.Query().Get("customer"))

	var monitors []models.MonitorListItem
	var err error
	if isPlatformAdmin(user) {
		if customerFilter != "" {
			monitors, err = s.store.ListMonitorsByTenant(customerFilter)
		} else {
			monitors, err = s.store.ListMonitors()
		}
	} else if user.TenantID != "" {
		monitors, err = s.store.ListMonitorsByTenant(user.TenantID)
	} else {
		monitors = []models.MonitorListItem{}
	}
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if tag != "" {
		monitors, err = filterMonitorsByTag(monitors, tag)
		if err != nil {
			jsonInternal(w, err)
			return
		}
	}
	if monitors == nil {
		monitors = []models.MonitorListItem{}
	}
	for i := range monitors {
		redactMonitor(&monitors[i].Monitor, user)
	}
	jsonOK(w, monitors)
}

func filterMonitorsByTag(items []models.MonitorListItem, tag string) ([]models.MonitorListItem, error) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	var filtered []models.MonitorListItem
	for _, item := range items {
		for _, t := range item.Tags {
			if strings.ToLower(t) == tag {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered, nil
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	m, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if m == nil || !canAccessTenant(user, m.TenantID) || (!isPlatformAdmin(user) && m.TenantID == "") {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	redactMonitor(m, user)
	jsonOK(w, m)
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	m.Enabled = true
	if isCustomerAdmin(user) {
		m.TenantID = user.TenantID
	} else if !isPlatformAdmin(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	} else {
		m.TenantID = strings.TrimSpace(m.TenantID)
	}

	if m.TenantID != "" {
		if err := s.store.AssertMonitorQuota(m.TenantID); err != nil {
			jsonError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if m.IntervalSeconds < 30 {
		m.IntervalSeconds = 30
	}
	if err := safehost.ValidateMonitorTarget(string(m.Type), m.URL, m.Port); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if m.Type == models.MonitorHeartbeat {
		token, err := randomToken(24)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "token error")
			return
		}
		m.HeartbeatToken = token
		if m.Name == "" {
			m.Name = "Heartbeat Monitor"
		}
	} else {
		m.HeartbeatToken = ""
	}
	if m.Tags == nil {
		m.Tags = []string{}
	}
	if err := s.store.CreateMonitor(&m); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "create", "monitor", m.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, m)
}

func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	existing, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if isCustomerAdmin(user) {
		if existing.TenantID != user.TenantID {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
	} else if !isPlatformAdmin(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}

	var input models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	prevTenant := existing.TenantID
	if input.Type != "" {
		existing.Type = input.Type
	}
	existing.Name = input.Name
	existing.URL = input.URL
	existing.Port = input.Port
	existing.Config = input.Config
	existing.Method = input.Method
	existing.ExpectedStatus = input.ExpectedStatus
	existing.ExpectedStatusMin = input.ExpectedStatusMin
	existing.ExpectedStatusMax = input.ExpectedStatusMax
	existing.KeywordMustExist = input.KeywordMustExist
	existing.KeywordMustNotExist = input.KeywordMustNotExist
	existing.RequestBody = input.RequestBody
	existing.RequestHeaders = input.RequestHeaders
	if input.IntervalSeconds < 30 {
		existing.IntervalSeconds = 30
	} else {
		existing.IntervalSeconds = input.IntervalSeconds
	}
	existing.TimeoutMs = input.TimeoutMs
	existing.SlowThresholdMs = input.SlowThresholdMs
	existing.FollowRedirects = input.FollowRedirects
	existing.AlertEmails = input.AlertEmails
	existing.Enabled = input.Enabled
	existing.Invert = input.Invert
	if input.AlertAfterFailures > 0 {
		existing.AlertAfterFailures = input.AlertAfterFailures
	}
	if input.Tags != nil {
		existing.Tags = input.Tags
	}
	if input.Config != "" {
		existing.Config = input.Config
	}

	if isCustomerAdmin(user) {
		existing.TenantID = user.TenantID
	} else {
		existing.TenantID = strings.TrimSpace(input.TenantID)
	}

	if err := safehost.ValidateMonitorTarget(string(existing.Type), existing.URL, existing.Port); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Quota when newly assigning or changing customer.
	if existing.TenantID != "" && existing.TenantID != prevTenant {
		if err := s.store.AssertMonitorQuota(existing.TenantID); err != nil {
			jsonError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if err := s.store.UpdateMonitor(existing); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "update", "monitor", existing.Name)
	jsonOK(w, existing)
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	existing, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if isCustomerAdmin(user) {
		if existing.TenantID != user.TenantID {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
	} else if !isPlatformAdmin(user) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.store.DeleteMonitor(id); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(user.Username, "delete", "monitor", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListResults(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	m, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if m == nil || !canAccessTenant(user, m.TenantID) || (!isPlatformAdmin(user) && m.TenantID == "") {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	limit := queryInt(r, "limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := queryInt(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	results, err := s.store.ListCheckResults(id, limit, offset)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if results == nil {
		results = []models.CheckResult{}
	}
	total, err := s.store.CountCheckResults(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, map[string]any{
		"items":  results,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) handleListMonitorIncidents(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	m, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if m == nil || !canAccessTenant(user, m.TenantID) || (!isPlatformAdmin(user) && m.TenantID == "") {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	limit := queryInt(r, "limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := queryInt(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	items, err := s.store.ListIncidentsByMonitor(id, limit, offset)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if items == nil {
		items = []models.IncidentListItem{}
	}
	total, err := s.store.CountIncidentsByMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id := r.PathValue("id")
	m, err := s.store.GetMonitor(id)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if m == nil || !canAccessTenant(user, m.TenantID) || (!isPlatformAdmin(user) && m.TenantID == "") {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	period := r.URL.Query().Get("period")
	var since time.Time
	switch period {
	case "7d":
		since = time.Now().AddDate(0, 0, -7)
	case "30d":
		since = time.Now().AddDate(0, 0, -30)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}
	stats, err := s.store.GetMonitorStats(id, since)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, stats)
}

func (s *Server) handleGetSMTP(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetSMTPConfig(s.defaultSMTP)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	cfg.Password = maskPassword(cfg.Password)
	jsonOK(w, cfg)
}

func (s *Server) handlePutSMTP(w http.ResponseWriter, r *http.Request) {
	var input models.SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if input.Password == "" || strings.HasPrefix(input.Password, "****") {
		existing, _ := s.store.GetSMTPConfig(s.defaultSMTP)
		input.Password = existing.Password
	}
	if input.Host != "" && input.Password == "" {
		jsonError(w, http.StatusBadRequest, "SMTP password is required")
		return
	}
	if err := s.store.SaveSMTPConfig(input); err != nil {
		jsonInternal(w, err)
		return
	}
	s.alerter.UpdateSMTP(input)
	input.Password = maskPassword(input.Password)
	jsonOK(w, input)
}

type testSMTPRequest struct {
	To string `json:"to"`
}

func (s *Server) handleTestSMTP(w http.ResponseWriter, r *http.Request) {
	var req testSMTPRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := s.alerter.SendTestEmail(req.To); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}
