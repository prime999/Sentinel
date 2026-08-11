package alerter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

var sslExpiryThresholds = []int{30, 7, 3, 1}

func (a *Alerter) HandleExtras(m *models.Monitor, result *models.CheckResult) error {
	switch m.Type {
	case models.MonitorSSL:
		return a.handleSSLExtras(m, result)
	case models.MonitorDNS:
		return a.handleDNSExtras(m, result)
	}
	return nil
}

func (a *Alerter) handleSSLExtras(m *models.Monitor, result *models.CheckResult) error {
	if result.Details == "" {
		return nil
	}
	var details models.SSLDetails
	if err := json.Unmarshal([]byte(result.Details), &details); err != nil {
		return nil
	}

	// Clear expiry warnings once the cert is healthy again (>30 days).
	if result.Status == models.StatusUp || result.Status == models.StatusDegraded || result.Status == models.StatusDown {
		if details.DaysRemaining > 30 {
			_ = a.store.ResolveOpenIncidents(m.ID, models.IncidentSSLExpiry, result.CheckedAt)
			_ = a.store.SaveLastAlertedSSLDays(m.ID, 0)
		}
		// Leftover open cert-change rows from older versions are notices, not outages.
		_ = a.store.ResolveOpenIncidents(m.ID, models.IncidentCertChange, result.CheckedAt)
	}

	// Certificate fingerprint change — informational event (renewal), not an outage.
	prevFP, _ := a.store.GetSnapshot(m.ID, "ssl_fp")
	if prevFP != "" && prevFP != details.Fingerprint {
		msg := fmt.Sprintf("Certificate fingerprint changed (issuer: %s)", details.Issuer)
		resolved := result.CheckedAt
		_ = a.store.CreateIncident(&models.Incident{
			MonitorID:  m.ID,
			Type:       models.IncidentCertChange,
			Message:    msg,
			StartedAt:  result.CheckedAt,
			ResolvedAt: &resolved,
		})
		_ = a.NotifyMonitor(m, "CERT CHANGE", msg, result.ResponseTimeMs)
	}
	_ = a.store.SaveSnapshot(m.ID, "ssl_fp", details.Fingerprint)

	// Expiry threshold alerts
	lastAlerted, _ := a.store.GetLastAlertedSSLDays(m.ID)
	for _, threshold := range sslExpiryThresholds {
		if details.DaysRemaining <= threshold && details.DaysRemaining >= 0 {
			if lastAlerted == 0 || threshold < lastAlerted {
				msg := fmt.Sprintf("SSL certificate for %s expires in %d days (%s)", m.URL, details.DaysRemaining, details.ExpiresAt)
				_ = a.store.CreateIncident(&models.Incident{
					MonitorID: m.ID, Type: models.IncidentSSLExpiry,
					Message: msg, StartedAt: result.CheckedAt,
				})
				_ = a.store.SaveLastAlertedSSLDays(m.ID, threshold)
				return a.NotifyMonitor(m, fmt.Sprintf("SSL EXPIRY (%dd)", threshold), msg, result.ResponseTimeMs)
			}
			break
		}
	}
	return nil
}

func (a *Alerter) handleDNSExtras(m *models.Monitor, result *models.CheckResult) error {
	if result.Details == "" {
		return nil
	}
	var details models.DNSDetails
	if err := json.Unmarshal([]byte(result.Details), &details); err != nil {
		return nil
	}
	if len(details.Changes) == 0 {
		return nil
	}

	// Cooldown: one DNS change alert per hour
	lastRaw, _ := a.store.GetSnapshot(m.ID, "dns_alert")
	if lastRaw != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastRaw); err == nil && time.Since(t) < time.Hour {
			return nil
		}
	}

	var msgs []string
	for _, ch := range details.Changes {
		msgs = append(msgs, fmt.Sprintf("%s record changed from %s to %s", ch.Type, ch.Before, ch.After))
	}
	msg := strings.Join(msgs, "; ")

	resolved := result.CheckedAt
	_ = a.store.CreateIncident(&models.Incident{
		MonitorID:  m.ID,
		Type:       models.IncidentDNSChange,
		Message:    msg,
		StartedAt:  result.CheckedAt,
		ResolvedAt: &resolved,
	})
	_ = a.store.SaveSnapshot(m.ID, "dns_alert", time.Now().UTC().Format(time.RFC3339Nano))
	return a.NotifyMonitor(m, "DNS CHANGE", msg, result.ResponseTimeMs)
}
