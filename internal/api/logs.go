package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sentinel-monitoring/sentinel/internal/logpath"
	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

const maxLogSourcesPerHost = 20
const maxLogAlertRulesPerHost = 50
const maxPatternLen = 200

var (
	globalLogRateMu sync.Mutex
	globalLogTokens = float64(1000)
	globalLogLast   = time.Now()
)

type tailLeaseEntry struct {
	SourceID string
	Until    time.Time
}

// SetLogsDB attaches the high-churn logs database.
func (s *Server) SetLogsDB(db *store.LogsDB) {
	s.logs = db
}

func (s *Server) logsDB() *store.LogsDB {
	return s.logs
}

func (s *Server) buildAgentConfig(h *models.Host) models.HostAgentConfig {
	cfg := h.AgentConfig()
	settings, err := s.store.GetLogSettings()
	if err != nil {
		settings = models.DefaultLogSettings()
	}
	cfg.MaxEventsPerSecHost = settings.MaxEventsPerSecHost
	cfg.MaxEventSizeBytes = settings.MaxEventSizeBytes
	cfg.MaxBatchEvents = settings.MaxBatchEvents
	sources, err := s.store.AgentLogSources(h.ID, settings)
	if err == nil {
		cfg.LogSources = sources
	}
	cfg.TailLeases = s.tailLeasesForHost(h.ID)
	return cfg
}

func (s *Server) tailLeasesForHost(hostID string) []models.AgentTailLease {
	s.tailMu.Lock()
	defer s.tailMu.Unlock()
	now := time.Now().UTC()
	var out []models.AgentTailLease
	leases := s.tailLeases[hostID]
	var keep []tailLeaseEntry
	for _, l := range leases {
		if l.Until.After(now) {
			keep = append(keep, l)
			out = append(out, models.AgentTailLease{
				SourceID: l.SourceID,
				Until:    l.Until.Format(time.RFC3339),
			})
		}
	}
	if len(keep) == 0 {
		delete(s.tailLeases, hostID)
	} else {
		s.tailLeases[hostID] = keep
	}
	if out == nil {
		out = []models.AgentTailLease{}
	}
	return out
}

func (s *Server) grantTailLease(hostID, sourceID string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	if ttl > 15*time.Minute {
		ttl = 15 * time.Minute
	}
	until := time.Now().UTC().Add(ttl)
	s.tailMu.Lock()
	defer s.tailMu.Unlock()
	if s.tailLeases == nil {
		s.tailLeases = map[string][]tailLeaseEntry{}
	}
	list := s.tailLeases[hostID]
	found := false
	for i := range list {
		if list[i].SourceID == sourceID {
			list[i].Until = until
			found = true
			break
		}
	}
	if !found {
		list = append(list, tailLeaseEntry{SourceID: sourceID, Until: until})
	}
	s.tailLeases[hostID] = list
}

func (s *Server) revokeTailLease(hostID, sourceID string) {
	s.tailMu.Lock()
	defer s.tailMu.Unlock()
	list := s.tailLeases[hostID]
	var keep []tailLeaseEntry
	for _, l := range list {
		if l.SourceID != sourceID {
			keep = append(keep, l)
		}
	}
	if len(keep) == 0 {
		delete(s.tailLeases, hostID)
	} else {
		s.tailLeases[hostID] = keep
	}
}

func validateLogPattern(pat string) error {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return fmt.Errorf("pattern required")
	}
	if len(pat) > maxPatternLen {
		return fmt.Errorf("pattern too long (max %d)", maxPatternLen)
	}
	if _, err := regexp.Compile(pat); err != nil {
		return fmt.Errorf("invalid pattern: %v", err)
	}
	return nil
}

func validateLogSourceInput(src *models.LogSource) error {
	src.Name = strings.TrimSpace(src.Name)
	src.Path = strings.TrimSpace(src.Path)
	if src.Name == "" {
		return fmt.Errorf("name required")
	}
	if src.Type == "" {
		src.Type = models.LogSourceFile
	}
	switch src.Type {
	case models.LogSourceFile:
		if err := logpath.ValidatePath(src.Path); err != nil {
			return err
		}
	case models.LogSourceJournal:
		unit := strings.TrimSpace(src.Path)
		if unit == "" {
			return fmt.Errorf("journal unit required")
		}
		if !models.JournalUnitAllowed(unit) {
			return fmt.Errorf("path not allowed: journal unit must be on the allowlist (e.g. nginx.service, php*-fpm.service)")
		}
		src.Path = unit
	default:
		return fmt.Errorf("unsupported source type")
	}
	if src.MinLevel == "" {
		src.MinLevel = models.LogLevelWarn
	}
	switch src.MinLevel {
	case models.LogLevelDebug, models.LogLevelInfo, models.LogLevelWarn, models.LogLevelError, models.LogLevelCrit:
	default:
		return fmt.Errorf("invalid min_level")
	}
	if src.IncludePatterns == nil {
		src.IncludePatterns = []string{}
	}
	for _, p := range src.IncludePatterns {
		if len(p) > maxPatternLen {
			return fmt.Errorf("include pattern too long")
		}
	}
	if src.Tags == nil {
		src.Tags = []string{}
	}
	return nil
}

func (s *Server) handleListLogSources(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	list, err := s.store.ListLogSources(h.ID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, list)
}

func (s *Server) handleCreateLogSource(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	n, err := s.store.CountLogSourcesByHost(h.ID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if n >= maxLogSourcesPerHost {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("max %d log sources per host", maxLogSourcesPerHost))
		return
	}
	var src models.LogSource
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&src); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	src.HostID = h.ID
	src.Enabled = true
	src.Health = models.LogHealthOK
	if err := validateLogSourceInput(&src); err != nil {
		msg := err.Error()
		if errors.Is(err, logpath.ErrRejectedPolicy) {
			msg = "Path not allowed: Sentinel can only monitor approved log locations"
		}
		jsonError(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.CreateLogSource(&src); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "create", "log_source", src.Path+" on "+h.DisplayName())
	jsonOK(w, src)
}

func (s *Server) handleUpdateLogSource(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	existing, err := s.store.GetLogSource(r.PathValue("sid"))
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || existing.HostID != h.ID {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	var src models.LogSource
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&src); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	src.ID = existing.ID
	src.HostID = h.ID
	src.CreatedAt = existing.CreatedAt
	src.DroppedLines = existing.DroppedLines
	src.Health = existing.Health
	src.HealthDetail = existing.HealthDetail
	if err := validateLogSourceInput(&src); err != nil {
		msg := err.Error()
		if errors.Is(err, logpath.ErrRejectedPolicy) {
			msg = "Path not allowed: Sentinel can only monitor approved log locations"
		}
		jsonError(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.UpdateLogSource(&src); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "update", "log_source", src.Path+" on "+h.DisplayName())
	jsonOK(w, src)
}

func (s *Server) handleDeleteLogSource(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	existing, err := s.store.GetLogSource(r.PathValue("sid"))
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || existing.HostID != h.ID {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.store.DeleteLogSource(existing.ID); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "delete", "log_source", existing.Path+" on "+h.DisplayName())
	jsonOK(w, map[string]bool{"ok": true})
}

func (s *Server) handleListLogAlertRules(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	list, err := s.store.ListLogAlertRules(h.ID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, list)
}

func (s *Server) handleCreateLogAlertRule(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	rules, err := s.store.ListLogAlertRules(h.ID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if len(rules) >= maxLogAlertRulesPerHost {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("max %d log alert rules per host", maxLogAlertRulesPerHost))
		return
	}
	var rule models.LogAlertRule
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&rule); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	rule.HostID = h.ID
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Pattern = strings.TrimSpace(rule.Pattern)
	if rule.Name == "" {
		jsonError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := validateLogPattern(rule.Pattern); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if rule.SourceID != "" {
		src, err := s.store.GetLogSource(rule.SourceID)
		if err != nil {
			jsonInternal(w, err)
			return
		}
		if src == nil || src.HostID != h.ID {
			jsonError(w, http.StatusBadRequest, "invalid source_id")
			return
		}
	}
	rule.Enabled = true
	if err := s.store.CreateLogAlertRule(&rule); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "create", "log_alert_rule", rule.Name+" on "+h.DisplayName())
	jsonOK(w, rule)
}

func (s *Server) handleUpdateLogAlertRule(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	existing, err := s.store.GetLogAlertRule(r.PathValue("rid"))
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || existing.HostID != h.ID {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	var rule models.LogAlertRule
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&rule); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	rule.ID = existing.ID
	rule.HostID = h.ID
	rule.CreatedAt = existing.CreatedAt
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Pattern = strings.TrimSpace(rule.Pattern)
	if rule.Name == "" {
		jsonError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := validateLogPattern(rule.Pattern); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if rule.SourceID != "" {
		src, err := s.store.GetLogSource(rule.SourceID)
		if err != nil {
			jsonInternal(w, err)
			return
		}
		if src == nil || src.HostID != h.ID {
			jsonError(w, http.StatusBadRequest, "invalid source_id")
			return
		}
	}
	if err := s.store.UpdateLogAlertRule(&rule); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "update", "log_alert_rule", rule.Name+" on "+h.DisplayName())
	jsonOK(w, rule)
}

func (s *Server) handleDeleteLogAlertRule(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	existing, err := s.store.GetLogAlertRule(r.PathValue("rid"))
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if existing == nil || existing.HostID != h.ID {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.store.DeleteLogAlertRule(existing.ID); err != nil {
		jsonInternal(w, err)
		return
	}
	_ = s.store.InsertAudit(currentUser(r).Username, "delete", "log_alert_rule", existing.Name+" on "+h.DisplayName())
	jsonOK(w, map[string]bool{"ok": true})
}

func (s *Server) handleQueryLogEvents(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	logs := s.logsDB()
	if logs == nil {
		jsonOK(w, []models.LogEvent{})
		return
	}
	q := store.LogEventQuery{
		HostID:   h.ID,
		SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
		Level:    strings.TrimSpace(r.URL.Query().Get("level")),
		Search:   strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:    100,
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &q.Limit)
	}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q.From = &t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q.To = &t
		}
	}
	events, err := logs.QueryEvents(q)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if events == nil {
		events = []models.LogEvent{}
	}
	jsonOK(w, events)
}

func (s *Server) handleQueryLogVolume(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	logs := s.logsDB()
	if logs == nil {
		jsonOK(w, []models.LogVolumeBucket{})
		return
	}
	to := time.Now().UTC()
	from := to.Add(-24 * time.Hour)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	sourceID := strings.TrimSpace(r.URL.Query().Get("source_id"))
	buckets, err := logs.QueryVolume(h.ID, sourceID, from, to)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if buckets == nil {
		buckets = []models.LogVolumeBucket{}
	}
	jsonOK(w, buckets)
}

func (s *Server) handleLogSourceTemplates(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, models.DefaultLogSourceTemplates())
}

func (s *Server) handleGetLogSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetLogSettings()
	if err != nil {
		jsonInternal(w, err)
		return
	}
	jsonOK(w, cfg)
}

func (s *Server) handlePutLogSettings(w http.ResponseWriter, r *http.Request) {
	var cfg models.LogSettings
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := s.store.SaveLogSettings(cfg); err != nil {
		jsonInternal(w, err)
		return
	}
	saved, _ := s.store.GetLogSettings()
	_ = s.store.InsertAudit(currentUser(r).Username, "update", "log_settings", "")
	jsonOK(w, saved)
}

func (s *Server) handleStartLogTail(w http.ResponseWriter, r *http.Request) {
	h, ok := s.loadVisibleHost(w, r)
	if !ok {
		return
	}
	src, err := s.store.GetLogSource(r.PathValue("sid"))
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if src == nil || src.HostID != h.ID {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}
	s.grantTailLease(h.ID, src.ID, 10*time.Minute)

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fmt.Fprintf(w, "event: lease\ndata: {\"source_id\":%q,\"until\":\"%s\"}\n\n", src.ID, time.Now().UTC().Add(10*time.Minute).Format(time.RFC3339))
	flusher.Flush()

	logs := s.logsDB()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	ctx := r.Context()
	seen := map[string]bool{}
	for {
		select {
		case <-ctx.Done():
			s.revokeTailLease(h.ID, src.ID)
			return
		case <-ticker.C:
			s.grantTailLease(h.ID, src.ID, 10*time.Minute) // heartbeat extends lease
			if logs == nil {
				continue
			}
			from := time.Now().UTC().Add(-2 * time.Minute)
			events, err := logs.QueryEvents(store.LogEventQuery{
				HostID: h.ID, SourceID: src.ID, From: &from, Limit: 50,
			})
			if err != nil {
				continue
			}
			for i := len(events) - 1; i >= 0; i-- {
				e := events[i]
				if seen[e.ID] {
					continue
				}
				seen[e.ID] = true
				b, _ := json.Marshal(e)
				fmt.Fprintf(w, "event: log\ndata: %s\n\n", b)
				flusher.Flush()
			}
			if len(seen) > 500 {
				seen = map[string]bool{}
			}
		}
	}
}

func (s *Server) handleAgentLogIngest(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.limits.Allow("host-logs:"+hashPrefix(token), 300, time.Minute) {
		jsonError(w, http.StatusTooManyRequests, "too many requests")
		return
	}
	h, err := s.store.GetHostByIngestToken(token)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	if h == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	settings, err := s.store.GetLogSettings()
	if err != nil {
		settings = models.DefaultLogSettings()
	}
	logs := s.logsDB()
	if logs == nil {
		jsonError(w, http.StatusServiceUnavailable, "logs store unavailable")
		return
	}
	if err := logs.EnsureUnderSizeCap(settings.MaxDBSizeBytes, settings.RetentionDays); err != nil {
		jsonError(w, http.StatusInsufficientStorage, "log store full")
		return
	}

	var batch models.LogIngestBatch
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&batch); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if len(batch.Events) > settings.MaxBatchEvents {
		batch.Events = batch.Events[:settings.MaxBatchEvents]
	}
	if !allowGlobalLogRate(settings.MaxEventsPerSecGlobal, len(batch.Events)) {
		jsonError(w, http.StatusTooManyRequests, "global log rate exceeded")
		return
	}

	sources, err := s.store.ListLogSources(h.ID)
	if err != nil {
		jsonInternal(w, err)
		return
	}
	byID := map[string]models.LogSource{}
	for _, src := range sources {
		byID[src.ID] = src
	}

	now := time.Now().UTC()
	var events []models.LogEvent
	hostRate := 0
	for _, e := range batch.Events {
		src, ok := byID[e.SourceID]
		if !ok || !src.Enabled {
			continue
		}
		if hostRate >= settings.MaxEventsPerSecHost*2 { // soft burst window
			break
		}
		msg := e.Message
		if !utf8.ValidString(msg) {
			continue
		}
		if len(msg) > settings.MaxEventSizeBytes {
			msg = msg[:settings.MaxEventSizeBytes]
		}
		ts := now
		if e.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				ts = t.UTC()
			}
		}
		level := models.LogLevel(strings.ToUpper(strings.TrimSpace(e.Level)))
		if level == "" {
			level = models.LogLevelInfo
		}
		fp := e.Fingerprint
		if fp == "" {
			fp = fingerprintMessage(msg)
		}
		events = append(events, models.LogEvent{
			HostID: h.ID, SourceID: e.SourceID, Timestamp: ts, Level: level,
			Message: msg, Fingerprint: fp, Metadata: e.Metadata, ReceivedAt: now,
		})
		hostRate++
	}
	if err := logs.InsertEventsBatch(events); err != nil {
		jsonInternal(w, err)
		return
	}

	var volumes []models.LogVolumeBucket
	for _, v := range batch.Volumes {
		if _, ok := byID[v.SourceID]; !ok {
			continue
		}
		start := now.Truncate(time.Minute)
		if v.BucketStart != "" {
			if t, err := time.Parse(time.RFC3339, v.BucketStart); err == nil {
				start = t.UTC().Truncate(time.Minute)
			}
		}
		volumes = append(volumes, models.LogVolumeBucket{
			HostID: h.ID, SourceID: v.SourceID, BucketStart: start,
			Level: models.LogLevel(strings.ToUpper(v.Level)), Count: v.Count,
		})
	}
	if err := logs.UpsertVolumeBatch(volumes); err != nil {
		log.Printf("log volume upsert: %v", err)
	}

	for _, health := range batch.Health {
		if _, ok := byID[health.SourceID]; !ok {
			continue
		}
		_ = s.store.UpdateLogSourceHealth(health.SourceID, models.LogSourceHealth(health.Health), health.Detail, health.DroppedLines)
	}

	rules, err := s.store.ListEnabledLogAlertRules(h.ID)
	if err == nil && len(rules) > 0 {
		if err := s.alerter.HandleLogAlertRules(h, logs, rules); err != nil {
			log.Printf("log alert eval: %v", err)
		}
	}

	fresh, _ := s.store.GetHost(h.ID)
	if fresh == nil {
		fresh = h
	}
	jsonOK(w, agentIngestResponse{OK: true, Config: s.buildAgentConfig(fresh)})
}

func allowGlobalLogRate(maxPerSec, n int) bool {
	if maxPerSec < 1 {
		maxPerSec = 1000
	}
	globalLogRateMu.Lock()
	defer globalLogRateMu.Unlock()
	now := time.Now()
	elapsed := now.Sub(globalLogLast).Seconds()
	globalLogLast = now
	globalLogTokens += elapsed * float64(maxPerSec)
	cap := float64(maxPerSec * 2)
	if globalLogTokens > cap {
		globalLogTokens = cap
	}
	need := float64(n)
	if need > globalLogTokens {
		return false
	}
	globalLogTokens -= need
	return true
}

func fingerprintMessage(msg string) string {
	norm := regexp.MustCompile(`\d+`).ReplaceAllString(msg, "#")
	if len(norm) > 512 {
		norm = norm[:512]
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}
