package alerter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/store"
)

func TestNotifyMonitor_SlackFiresWhenEmailDisabled(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty body")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.SaveSlackConfig("", models.SlackConfig{
		WebhookURL: srv.URL,
		Enabled:    true,
		Events:     []string{"all"},
	}); err != nil {
		t.Fatal(err)
	}

	a := New(st, models.SMTPConfig{Enabled: false}, models.SMTPConfig{}, "http://localhost")
	m := &models.Monitor{ID: "m1", Name: "Site", URL: "https://example.com", Type: models.MonitorHTTP}
	_ = a.NotifyMonitor(m, "DOWN", "unreachable", 50)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && hits.Load() == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if hits.Load() != 1 {
		t.Fatalf("slack hits=%d want 1", hits.Load())
	}
}

func TestNotifyMonitor_TenantSlackScoping(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.SaveSlackConfig("tenant-a", models.SlackConfig{
		WebhookURL: srv.URL,
		Enabled:    true,
		Events:     []string{"all"},
	}); err != nil {
		t.Fatal(err)
	}

	a := New(st, models.SMTPConfig{Enabled: false}, models.SMTPConfig{}, "http://localhost")
	// Platform monitor should not hit tenant-a Slack.
	_ = a.NotifyMonitor(&models.Monitor{ID: "1", Name: "P", URL: "https://p.example", TenantID: ""}, "DOWN", "x", 1)
	time.Sleep(100 * time.Millisecond)
	if hits.Load() != 0 {
		t.Fatalf("platform alert hit tenant slack: %d", hits.Load())
	}

	_ = a.NotifyMonitor(&models.Monitor{ID: "2", Name: "C", URL: "https://c.example", TenantID: "tenant-a"}, "DOWN", "x", 1)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && hits.Load() == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if hits.Load() != 1 {
		t.Fatalf("tenant alert hits=%d want 1", hits.Load())
	}
}
