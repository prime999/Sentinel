package alerter

import (
	"testing"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func TestDefaultRecipientsUsesAlertEmails(t *testing.T) {
	a := &Alerter{cfg: models.SMTPConfig{AlertEmails: "ops@example.com", From: "alerts@example.com"}}
	got := a.defaultRecipients()
	if len(got) != 1 || got[0] != "ops@example.com" {
		t.Fatalf("defaultRecipients = %v, want ops@example.com", got)
	}
}

func TestDefaultRecipientsDoesNotUseFrom(t *testing.T) {
	a := &Alerter{cfg: models.SMTPConfig{From: "alerts@example.com"}}
	got := a.defaultRecipients()
	if len(got) != 0 {
		t.Fatalf("defaultRecipients = %v, want empty (From is sender only)", got)
	}
}

func TestRecipientsMonitorEmails(t *testing.T) {
	a := &Alerter{cfg: models.SMTPConfig{From: "from@example.com"}}
	m := &models.Monitor{AlertEmails: "a@x.com, b@x.com"}
	got := a.recipients(m)
	if len(got) != 2 {
		t.Fatalf("recipients = %v", got)
	}
}

func TestPerfRecipientsUsesMonitorThenDefault(t *testing.T) {
	a := &Alerter{cfg: models.SMTPConfig{AlertEmails: "ops@example.com", From: "alerts@example.com"}}
	t1 := &models.PerformanceTarget{AlertEmails: ""}
	got := a.perfRecipients(t1)
	if len(got) != 1 || got[0] != "ops@example.com" {
		t.Fatalf("perfRecipients = %v", got)
	}
	t2 := &models.PerformanceTarget{AlertEmails: "perf@example.com"}
	got = a.perfRecipients(t2)
	if len(got) != 1 || got[0] != "perf@example.com" {
		t.Fatalf("perfRecipients = %v", got)
	}
}
