package alerter

import (
	"fmt"
	"log"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

// HandleLogAlertRules evaluates pattern thresholds after a log ingest batch.
func (a *Alerter) HandleLogAlertRules(h *models.Host, logs *store.LogsDB, rules []models.LogAlertRule) error {
	if a == nil || h == nil || logs == nil || !h.Enabled {
		return nil
	}
	now := time.Now().UTC()
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		window := time.Duration(rule.WindowSeconds) * time.Second
		if window < time.Minute {
			window = 5 * time.Minute
		}
		since := now.Add(-window)
		count, err := logs.CountEventsSince(h.ID, rule.SourceID, rule.Pattern, since)
		if err != nil {
			return err
		}
		open, err := a.store.GetOpenLogRuleIncident(h.ID, rule.ID)
		if err != nil {
			return err
		}
		if count >= rule.Threshold {
			if open != nil {
				continue
			}
			msg := fmt.Sprintf("%s — %d matching entries in the last %s on %s.",
				rule.Name, count, formatWindow(rule.WindowSeconds), h.DisplayName())
			inc := &models.Incident{
				MonitorID: h.ID,
				Type:      models.IncidentHostLog,
				Message:   msg,
				StartedAt: now,
				Details:   store.LogRuleDetailsJSON(rule.ID),
			}
			if err := a.store.CreateIncident(inc); err != nil {
				return err
			}
			notifyHost := *h
			notifyHost.NotifyEmail = rule.NotifyEmail
			notifyHost.NotifySlack = rule.NotifySlack
			notifyHost.NotifyWebhooks = rule.NotifyWebhooks
			event := "HOST_LOG"
			if rule.Severity == "critical" {
				event = "HOST_LOG_CRITICAL"
			}
			if err := hostNotify(a, &notifyHost, AlertMeta{
				Event:      event,
				Message:    msg,
				IncidentID: inc.ID,
				EventAt:    now,
			}); err != nil {
				log.Printf("alerter: host log notify: %v", err)
			}
			continue
		}
		// Cool down: resolve when count drops below half threshold (or zero).
		cool := rule.Threshold / 2
		if cool < 1 {
			cool = 0
		}
		if open != nil && count <= cool {
			_ = a.store.ResolveOpenLogRuleIncident(h.ID, rule.ID, now)
			started := open.StartedAt
			notifyHost := *h
			notifyHost.NotifyEmail = rule.NotifyEmail
			notifyHost.NotifySlack = rule.NotifySlack
			notifyHost.NotifyWebhooks = rule.NotifyWebhooks
			_ = hostNotify(a, &notifyHost, AlertMeta{
				Event:          "RECOVERY",
				Message:        "Recovered: " + rule.Name,
				IncidentID:     open.ID,
				EventAt:        now,
				StartedAt:      &started,
				RecoveredLabel: "Log alert cleared",
			})
		}
	}
	return nil
}

func formatWindow(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}
