package alerter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

func TestHandleResult_OneDownEmailPerOutage(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	m := &models.Monitor{
		Name:               "site",
		Type:               models.MonitorHTTP,
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutMs:          5000,
		SlowThresholdMs:    3000,
		Enabled:            true,
		AlertAfterFailures: 2,
		LastStatus:         models.StatusUp,
	}
	if err := st.CreateMonitor(m); err != nil {
		t.Fatal(err)
	}

	var alerts []string
	a := New(st, models.SMTPConfig{}, models.SMTPConfig{}, "http://localhost")
	a.notifyHook = func(_ *models.Monitor, alertType, _ string, _ int) error {
		alerts = append(alerts, alertType)
		return nil
	}

	now := time.Now().UTC()
	down := func(i int) *models.CheckResult {
		return &models.CheckResult{
			MonitorID: m.ID,
			Status:    models.StatusDown,
			Error:     "timeout",
			CheckedAt: now.Add(time.Duration(i) * time.Minute),
		}
	}
	up := func(i int) *models.CheckResult {
		return &models.CheckResult{
			MonitorID: m.ID,
			Status:    models.StatusUp,
			CheckedAt: now.Add(time.Duration(i) * time.Minute),
		}
	}

	// Need 2 consecutive failures before DOWN.
	if err := a.HandleResult(m, down(1)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("alerts=%v, want none before threshold", alerts)
	}

	if err := a.HandleResult(m, down(2)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0] != "DOWN" {
		t.Fatalf("alerts=%v, want [DOWN]", alerts)
	}

	// Long outage with more failures — no extra DOWN emails.
	for i := 3; i <= 20; i++ {
		if err := a.HandleResult(m, down(i)); err != nil {
			t.Fatal(err)
		}
	}
	if len(alerts) != 1 {
		t.Fatalf("alerts=%v, want a single DOWN during outage", alerts)
	}

	// One brief up is not enough for RECOVERY (threshold=2).
	if err := a.HandleResult(m, up(21)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("alerts=%v, want still only DOWN after one up", alerts)
	}
	if m.LastStatus != models.StatusDown {
		t.Fatalf("last_status=%s, want down while recovery pending", m.LastStatus)
	}

	// Flap back down — still no second DOWN email.
	if err := a.HandleResult(m, down(22)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("alerts=%v, want still one DOWN after flap", alerts)
	}

	// Confirmed recovery: 2 consecutive ups.
	if err := a.HandleResult(m, up(23)); err != nil {
		t.Fatal(err)
	}
	if err := a.HandleResult(m, up(24)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 || alerts[1] != "RECOVERY" {
		t.Fatalf("alerts=%v, want [DOWN RECOVERY]", alerts)
	}
	if m.LastStatus != models.StatusUp {
		t.Fatalf("last_status=%s, want up after recovery", m.LastStatus)
	}

	// A later outage may send DOWN again.
	if err := a.HandleResult(m, down(25)); err != nil {
		t.Fatal(err)
	}
	if err := a.HandleResult(m, down(26)); err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 3 || alerts[2] != "DOWN" {
		t.Fatalf("alerts=%v, want second outage DOWN", alerts)
	}
}
