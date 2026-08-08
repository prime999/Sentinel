package alerter

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

const defaultFlapThreshold = 2

func monitorAlertAfterFailures(m *models.Monitor) int {
	if m.AlertAfterFailures < 1 {
		return defaultFlapThreshold
	}
	return m.AlertAfterFailures
}

type Alerter struct {
	store        *store.Store
	cfg          models.SMTPConfig
	fallbackSMTP models.SMTPConfig
	dashboardURL string

	// recoveryStreak counts consecutive non-down checks while a DOWN incident is open.
	// Recovery email fires only after reaching the same threshold used for DOWN alerts,
	// so brief flaps during a long outage do not spam DOWN/RECOVERY pairs.
	mu             sync.Mutex
	recoveryStreak map[string]int

	// notifyHook, when set, replaces NotifyMonitor (tests).
	notifyHook func(m *models.Monitor, alertType, message string, responseMs int) error
}

func New(s *store.Store, cfg models.SMTPConfig, fallback models.SMTPConfig, dashboardURL string) *Alerter {
	a := &Alerter{
		store:          s,
		cfg:            cfg,
		fallbackSMTP:   fallback,
		dashboardURL:   dashboardURL,
		recoveryStreak: make(map[string]int),
	}
	a.refreshSMTP()
	return a
}

func (a *Alerter) notifyMonitorAlert(m *models.Monitor, alertType, message string, responseMs int) error {
	if a.notifyHook != nil {
		return a.notifyHook(m, alertType, message, responseMs)
	}
	return a.NotifyMonitor(m, alertType, message, responseMs)
}

func (a *Alerter) incRecoveryStreak(monitorID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.recoveryStreak == nil {
		a.recoveryStreak = make(map[string]int)
	}
	a.recoveryStreak[monitorID]++
	return a.recoveryStreak[monitorID]
}

func (a *Alerter) clearRecoveryStreak(monitorID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.recoveryStreak != nil {
		delete(a.recoveryStreak, monitorID)
	}
}

func (a *Alerter) UpdateSMTP(cfg models.SMTPConfig) {
	a.cfg = cfg
}

func (a *Alerter) refreshSMTP() {
	if a.store == nil {
		return
	}
	cfg, err := a.store.GetSMTPConfig(a.fallbackSMTP)
	if err != nil {
		log.Printf("alerter: load smtp config: %v", err)
		return
	}
	if cfg.Host != "" {
		a.cfg = cfg
	}
}

func (a *Alerter) HandleResult(m *models.Monitor, result *models.CheckResult) error {
	newStatus := result.Status
	failures := m.ConsecutiveFailures
	threshold := monitorAlertAfterFailures(m)

	open, err := a.store.GetOpenIncident(m.ID, models.IncidentDown)
	if err != nil {
		return err
	}

	if newStatus == models.StatusDown {
		failures++
		a.clearRecoveryStreak(m.ID)

		if err := a.store.UpdateMonitorState(m.ID, models.StatusDown, failures, result.CheckedAt); err != nil {
			return err
		}
		m.LastStatus = models.StatusDown
		m.ConsecutiveFailures = failures

		// Exactly one DOWN email per outage (while the incident stays open).
		if failures < threshold {
			return nil
		}
		if open != nil {
			log.Printf("alerter: down alert skipped for %s: incident already open", m.Name)
			return nil
		}
		inc := &models.Incident{
			MonitorID: m.ID,
			Type:      models.IncidentDown,
			Message:   result.Error,
			StartedAt: result.CheckedAt,
		}
		if err := a.store.CreateIncident(inc); err != nil {
			return err
		}
		return a.notifyMonitorAlert(m, "DOWN", result.Error, result.ResponseTimeMs)
	}

	// Non-down check
	if open != nil {
		streak := a.incRecoveryStreak(m.ID)
		if streak < threshold {
			// Brief up during an outage: keep showing Down and do not email yet.
			if err := a.store.UpdateMonitorState(m.ID, models.StatusDown, m.ConsecutiveFailures, result.CheckedAt); err != nil {
				return err
			}
			m.LastStatus = models.StatusDown
			log.Printf("alerter: recovery pending for %s: %d/%d successful checks", m.Name, streak, threshold)
			return nil
		}
		a.clearRecoveryStreak(m.ID)
		if err := a.store.UpdateMonitorState(m.ID, newStatus, 0, result.CheckedAt); err != nil {
			return err
		}
		m.LastStatus = newStatus
		m.ConsecutiveFailures = 0
		_ = a.store.ResolveOpenIncidents(m.ID, models.IncidentDown, result.CheckedAt)
		_ = a.store.CreateIncident(&models.Incident{
			MonitorID: m.ID,
			Type:      models.IncidentRecovery,
			Message:   "Monitor is back online",
			StartedAt: result.CheckedAt,
		})
		_ = a.store.ResolveOpenIncidents(m.ID, models.IncidentSlow, result.CheckedAt)
		return a.notifyMonitorAlert(m, "RECOVERY", "Monitor is back online", result.ResponseTimeMs)
	}

	a.clearRecoveryStreak(m.ID)
	if err := a.store.UpdateMonitorState(m.ID, newStatus, 0, result.CheckedAt); err != nil {
		return err
	}
	m.LastStatus = newStatus
	m.ConsecutiveFailures = 0

	// Uptime monitors do not email on latency/degraded — performance-target only.
	if newStatus != models.StatusDegraded {
		_ = a.store.ResolveOpenIncidents(m.ID, models.IncidentSlow, result.CheckedAt)
	}
	return nil
}

func (a *Alerter) SendPasswordResetEmail(to, username, resetURL string) error {
	if a.cfg.Host == "" {
		return fmt.Errorf("SMTP not configured — configure email in Settings")
	}
	if to == "" {
		return fmt.Errorf("no email address")
	}
	subject := "[Sentinel] Password Reset"
	body := a.renderPasswordResetEmail(username, resetURL)
	return a.sendSMTP(to, subject, body)
}

func (a *Alerter) SendPasswordChangedEmail(to, username string) error {
	if a.cfg.Host == "" || to == "" {
		return nil
	}
	subject := "[Sentinel] Password Changed"
	body := a.renderPasswordChangedEmail(username)
	return a.sendSMTP(to, subject, body)
}

func (a *Alerter) SendTestEmail(to string) error {
	if a.cfg.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}
	if to == "" {
		to = a.cfg.From
	}
	subject := "[Sentinel] Test Email"
	body := a.renderEmail("Test Alert", "Sentinel Monitoring", "https://example.com", "This is a test email from Sentinel.", "TEST", 0)
	return a.sendSMTP(to, subject, body)
}

func (a *Alerter) sendAlert(m *models.Monitor, alertType, message string, responseMs int) error {
	a.refreshSMTP()
	if a.cfg.Host == "" {
		return fmt.Errorf("SMTP not configured — set host in Settings → SMTP")
	}
	recipients := a.recipients(m)
	if len(recipients) == 0 {
		return fmt.Errorf("no alert recipients — set alert emails on the monitor, SMTP Alert Recipients, or a profile email")
	}
	subject := fmt.Sprintf("[Sentinel] %s: %s", alertType, m.Name)
	body := a.renderEmail(m.Name, m.URL, a.dashboardURL+"/monitors/"+m.ID, message, alertType, responseMs)
	for _, to := range recipients {
		if err := a.sendSMTP(to, subject, body); err != nil {
			return err
		}
	}
	return nil
}

func parseEmails(raw string) []string {
	var out []string
	for _, e := range strings.Split(raw, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}

func (a *Alerter) defaultRecipients() []string {
	if emails := parseEmails(a.cfg.AlertEmails); len(emails) > 0 {
		return emails
	}
	if a.store != nil {
		if emails, err := a.store.ListAlertProfileEmails(); err == nil && len(emails) > 0 {
			return emails
		}
	}
	return nil
}

func (a *Alerter) recipients(m *models.Monitor) []string {
	if emails := parseEmails(m.AlertEmails); len(emails) > 0 {
		return emails
	}
	return a.defaultRecipients()
}

func (a *Alerter) sendSMTP(to, subject, htmlBody string) error {
	a.refreshSMTP()
	if a.cfg.Host == "" {
		err := fmt.Errorf("SMTP not configured")
		log.Printf("smtp: SKIP to=%s subject=%q: %v", to, subject, err)
		return err
	}
	if to == "" {
		err := fmt.Errorf("empty recipient")
		log.Printf("smtp: SKIP subject=%q: %v", subject, err)
		return err
	}

	addr := fmt.Sprintf("%s:%d", a.cfg.Host, a.cfg.Port)
	from := a.cfg.From
	if from == "" {
		from = a.cfg.Username
	}
	if from == "" {
		err := fmt.Errorf("SMTP from address not configured")
		log.Printf("smtp: SKIP to=%s subject=%q: %v", to, subject, err)
		return err
	}

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", formatFromHeader(from)))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	msg.WriteString(htmlBody)

	auth := smtpAuth(a.cfg)

	var sendErr error
	if a.cfg.Port == 465 {
		sendErr = a.sendSMTPSImplicitTLS(addr, auth, from, to, msg.Bytes())
	} else {
		sendErr = a.sendSMTPStartTLS(addr, auth, from, to, msg.Bytes())
	}
	if sendErr != nil {
		log.Printf("smtp: FAIL to=%s from=%s subject=%q host=%s: %v", to, from, subject, addr, sendErr)
		return sendErr
	}
	log.Printf("smtp: SENT to=%s from=%s subject=%q host=%s", to, from, subject, addr)
	return nil
}

func smtpAuth(cfg models.SMTPConfig) smtp.Auth {
	if cfg.Username == "" {
		return nil
	}
	return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
}

// formatFromHeader sets a friendly display name for inbox UIs while keeping the
// envelope address as the bare email (MAIL FROM).
func formatFromHeader(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return from
	}
	// Already has a display name.
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		return from
	}
	return fmt.Sprintf("\"Sentinel Monitoring\" <%s>", from)
}

func (a *Alerter) tlsConfig() *tls.Config {
	return &tls.Config{
		ServerName: a.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}
}

func (a *Alerter) sendSMTPStartTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(a.tlsConfig()); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	} else if a.cfg.TLS {
		return fmt.Errorf("server does not support STARTTLS")
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return client.Quit()
}

func (a *Alerter) sendSMTPSImplicitTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, a.tlsConfig())
	if err != nil {
		return fmt.Errorf("TLS connect: %w", err)
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = a.cfg.Host
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return client.Quit()
}

var emailTmpl = template.Must(template.New("email").Parse(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;background:#f4f4f5;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:24px;border:1px solid #e4e4e7;">
  <h2 style="margin:0 0 8px;color:#18181b;">Sentinel Alert: {{.Type}}</h2>
  <p style="color:#71717a;margin:0 0 16px;">{{.Time}}</p>
  <table style="width:100%;border-collapse:collapse;">
    <tr><td style="padding:8px 0;color:#71717a;">Monitor</td><td style="padding:8px 0;font-weight:600;">{{.Name}}</td></tr>
    <tr><td style="padding:8px 0;color:#71717a;">URL</td><td style="padding:8px 0;"><a href="{{.URL}}">{{.URL}}</a></td></tr>
    {{if .ResponseMs}}<tr><td style="padding:8px 0;color:#71717a;">Response Time</td><td style="padding:8px 0;">{{.ResponseMs}} ms</td></tr>{{end}}
    <tr><td style="padding:8px 0;color:#71717a;">Message</td><td style="padding:8px 0;">{{.Message}}</td></tr>
  </table>
  <p style="margin-top:20px;"><a href="{{.DashboardURL}}" style="background:#2563eb;color:#fff;padding:10px 16px;border-radius:6px;text-decoration:none;">View Dashboard</a></p>
</div></body></html>`))

type emailData struct {
	Type, Name, URL, DashboardURL, Message, Time string
	ResponseMs int
}

var resetEmailTmpl = template.Must(template.New("reset").Parse(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;background:#f4f4f5;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:24px;border:1px solid #e4e4e7;">
  <h2 style="margin:0 0 8px;color:#18181b;">Reset your password</h2>
  <p style="color:#71717a;">Hi {{.Username}}, click the button below to set a new password. This link expires in 1 hour.</p>
  <p style="margin-top:20px;"><a href="{{.ResetURL}}" style="background:#2B7A78;color:#fff;padding:10px 16px;border-radius:6px;text-decoration:none;">Reset Password</a></p>
  <p style="color:#71717a;font-size:12px;margin-top:24px;">If you did not request this, you can ignore this email.</p>
</div></body></html>`))

func (a *Alerter) renderPasswordResetEmail(username, resetURL string) string {
	var buf bytes.Buffer
	_ = resetEmailTmpl.Execute(&buf, map[string]string{"Username": username, "ResetURL": resetURL})
	return buf.String()
}

var passwordChangedTmpl = template.Must(template.New("changed").Parse(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;background:#f4f4f5;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:24px;border:1px solid #e4e4e7;">
  <h2 style="margin:0 0 8px;color:#18181b;">Password changed successfully</h2>
  <p style="color:#71717a;">Hi {{.Username}}, your Sentinel account password was changed on {{.Time}}.</p>
  <p style="color:#71717a;">If you made this change, no further action is needed.</p>
  <p style="color:#71717a;font-size:13px;margin-top:20px;">If you did <strong>not</strong> change your password, contact your administrator immediately and sign in to update your credentials.</p>
  <p style="margin-top:20px;"><a href="{{.DashboardURL}}" style="background:#2B7A78;color:#fff;padding:10px 16px;border-radius:6px;text-decoration:none;">Open Sentinel</a></p>
</div></body></html>`))

func (a *Alerter) renderPasswordChangedEmail(username string) string {
	var buf bytes.Buffer
	_ = passwordChangedTmpl.Execute(&buf, map[string]string{
		"Username":      username,
		"Time":          time.Now().UTC().Format(time.RFC1123),
		"DashboardURL":  strings.TrimRight(a.dashboardURL, "/") + "/login",
	})
	return buf.String()
}

func (a *Alerter) renderEmail(name, url, dashboardURL, message, alertType string, responseMs int) string {
	var buf bytes.Buffer
	_ = emailTmpl.Execute(&buf, emailData{
		Type: alertType, Name: name, URL: url, DashboardURL: dashboardURL,
		Message: message, Time: time.Now().UTC().Format(time.RFC1123),
		ResponseMs: responseMs,
	})
	return buf.String()
}
