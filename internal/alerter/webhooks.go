package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (a *Alerter) inMaintenance(monitorID string) bool {
	ok, err := a.store.IsInMaintenance(monitorID, time.Now().UTC())
	return err == nil && ok
}

func (a *Alerter) fireWebhooks(event string, payload map[string]any) {
	hooks, err := a.store.GetWebhooks()
	if err != nil {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, hook := range hooks {
		if !hook.Enabled || hook.URL == "" {
			continue
		}
		if !webhookMatchesEvent(hook.Events, event) {
			continue
		}
		go func(url string) {
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
		}(hook.URL)
	}
}

func webhookMatchesEvent(events []string, event string) bool {
	if len(events) == 0 {
		return true
	}
	for _, e := range events {
		if strings.EqualFold(e, event) || strings.EqualFold(e, "all") {
			return true
		}
	}
	return false
}

func (a *Alerter) NotifyMonitor(m *models.Monitor, alertType, message string, responseMs int) error {
	if a.inMaintenance(m.ID) {
		log.Printf("alerter: %s alert skipped for %s: in maintenance", alertType, m.Name)
		return nil
	}
	if err := a.sendAlert(m, alertType, message, responseMs); err != nil {
		return err
	}
	a.fireWebhooks(alertType, map[string]any{
		"event":            alertType,
		"monitor_id":       m.ID,
		"monitor_name":     m.Name,
		"monitor_type":     m.Type,
		"url":              m.URL,
		"message":          message,
		"response_time_ms": responseMs,
		"dashboard_url":    a.dashboardURL + "/monitors/" + m.ID,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (a *Alerter) HandlePerformanceResult(t *models.PerformanceTarget, result *models.PerformanceResult, prevStatus models.MonitorStatus) error {
	if a.inMaintenance(t.ID) {
		return nil
	}

	prev := prevStatus
	newStatus := result.Status
	if newStatus == models.StatusDown {
		newStatus = models.StatusDegraded
	}
	if prev == models.StatusDown {
		prev = models.StatusDegraded
	}

	wasSlow := prev == models.StatusDegraded
	isSlow := newStatus == models.StatusDegraded

	if isSlow {
		openSlow, _ := a.store.GetOpenIncident(t.ID, models.IncidentSlow)
		if openSlow != nil {
			return nil
		}
		lastSlow, _ := a.store.GetLastSlowAlertAt(t.ID)
		if lastSlow != nil && time.Since(*lastSlow) < slowAlertCooldown {
			return nil
		}
		pct, total, slow, _ := a.store.GetPerformanceSlowStats(t.ID, time.Now().Add(-time.Hour))
		if total == 0 {
			pct = 100
			total = 1
			slow = 1
		}
		msg := fmt.Sprintf("%.1f%% of checks slow (%d of %d in the last hour)", pct, slow, total)
		_ = a.store.CreateIncident(&models.Incident{
			MonitorID: t.ID, Type: models.IncidentSlow, Message: msg, StartedAt: result.CheckedAt,
		})
		return a.sendPerformanceAlert(t, "SLOW", msg, result.ResponseTimeMs)
	}

	if !isSlow && wasSlow {
		_ = a.store.ResolveOpenIncidents(t.ID, models.IncidentSlow, result.CheckedAt)
		_ = a.store.ResolveOpenIncidents(t.ID, models.IncidentDown, result.CheckedAt)
		return a.sendPerformanceAlert(t, "NORMAL", "Back to normal", result.ResponseTimeMs)
	}

	return nil
}

func (a *Alerter) sendPerformanceAlert(t *models.PerformanceTarget, alertType, message string, responseMs int) error {
	a.refreshSMTP()
	payload := map[string]any{
		"event":            alertType,
		"target_id":        t.ID,
		"target_name":      t.Name,
		"url":              t.URL,
		"message":          message,
		"response_time_ms": responseMs,
		"dashboard_url":    a.dashboardURL + "/performance/" + t.ID,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}
	if a.cfg.Host == "" {
		a.fireWebhooks(alertType, payload)
		return fmt.Errorf("SMTP not configured")
	}
	recipients := a.perfRecipients(t)
	if len(recipients) == 0 {
		return fmt.Errorf("no alert recipients — set alert emails on the target, SMTP Alert Recipients, or a profile email")
	}
	subject := fmt.Sprintf("[Sentinel] %s: %s", alertType, t.Name)
	body := a.renderEmail(t.Name, t.URL, a.dashboardURL+"/performance/"+t.ID, message, alertType, responseMs)
	for _, to := range recipients {
		if err := a.sendSMTP(to, subject, body); err != nil {
			return err
		}
	}
	a.fireWebhooks(alertType, payload)
	return nil
}

func (a *Alerter) perfRecipients(t *models.PerformanceTarget) []string {
	if emails := parseEmails(t.AlertEmails); len(emails) > 0 {
		return emails
	}
	return a.defaultRecipients()
}
