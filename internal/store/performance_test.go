package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func TestListMonitorRowStats(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	a := &models.Monitor{Name: "a", URL: "https://a.example", Enabled: true}
	b := &models.Monitor{Name: "b", URL: "https://b.example", Enabled: true}
	if err := st.CreateMonitor(a); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateMonitor(b); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	since := now.Add(-30 * 24 * time.Hour)
	insert := func(monitorID string, i int, status models.MonitorStatus, rt int) {
		t.Helper()
		if err := st.InsertCheckResult(&models.CheckResult{
			MonitorID:      monitorID,
			Status:         status,
			ResponseTimeMs: rt,
			CheckedAt:      now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 26; i++ {
		status := models.StatusUp
		if i == 0 {
			status = models.StatusDown
		}
		insert(a.ID, i, status, 100+i)
	}
	insert(b.ID, 0, models.StatusUp, 50)
	insert(b.ID, 1, models.StatusDegraded, 80)

	got, err := st.ListMonitorRowStats([]string{a.ID, b.ID, "missing"}, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}

	full, err := st.GetMonitorStats(a.ID, since)
	if err != nil {
		t.Fatal(err)
	}
	row := got[a.ID]
	if row.UptimePct != full.UptimePct {
		t.Fatalf("a uptime=%v, want %v", row.UptimePct, full.UptimePct)
	}
	if len(row.Points) != 24 {
		t.Fatalf("a points=%d, want 24", len(row.Points))
	}
	if row.Points[0] != 102 || row.Points[23] != 125 {
		t.Fatalf("a points=%v, want last 24 latencies 102..125", row.Points)
	}

	if got[b.ID].UptimePct != 100 {
		t.Fatalf("b uptime=%v, want 100", got[b.ID].UptimePct)
	}
	if len(got[b.ID].Points) != 2 || got[b.ID].Points[0] != 50 || got[b.ID].Points[1] != 80 {
		t.Fatalf("b points=%v", got[b.ID].Points)
	}
	if got["missing"].UptimePct != 0 || len(got["missing"].Points) != 0 {
		t.Fatalf("missing=%+v", got["missing"])
	}
}
