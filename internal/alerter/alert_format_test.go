package alerter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAlertMetaLabels(t *testing.T) {
	started := time.Date(2026, 8, 11, 10, 13, 0, 0, time.UTC)
	meta := AlertMeta{
		Event:      "RECOVERY",
		Name:       "Google",
		IncidentID: "abcdef12-zzzz",
		ResponseMs: 168,
		EventAt:    started.Add(2*time.Minute + 14*time.Second),
		StartedAt:  &started,
	}
	if meta.Title() != "MONITOR RECOVERED" {
		t.Fatalf("title=%q", meta.Title())
	}
	if meta.DowntimeLabel() != "2m 14s" {
		t.Fatalf("downtime=%q", meta.DowntimeLabel())
	}
	if meta.IncidentLabel() != "INC-ABCDEF12" {
		t.Fatalf("incident=%q", meta.IncidentLabel())
	}
	if meta.ResponseLabel() != "168ms" {
		t.Fatalf("response=%q", meta.ResponseLabel())
	}
}

func TestAlertMetaTimeoutLabel(t *testing.T) {
	meta := AlertMeta{Event: "DOWN", Message: "context deadline exceeded", ResponseMs: 10000}
	if meta.ResponseLabel() != "Timeout" {
		t.Fatalf("response=%q", meta.ResponseLabel())
	}
}

func TestBuildSlackPayload(t *testing.T) {
	meta := AlertMeta{
		Event:        "DOWN",
		Name:         "Google",
		URL:          "https://www.google.com/",
		Message:      "connection refused",
		DashboardURL: "http://localhost/monitors/1",
		ResponseMs:   176,
		IncidentID:   "deadbeef",
		EventAt:      time.Date(2026, 8, 11, 16, 13, 0, 0, time.UTC),
	}
	raw, err := buildSlackPayload(meta)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload["text"].(string), "DOWN: Google") {
		t.Fatalf("fallback text=%v", payload["text"])
	}
	atts := payload["attachments"].([]any)
	if len(atts) != 1 {
		t.Fatalf("attachments=%d", len(atts))
	}
	att := atts[0].(map[string]any)
	if att["color"] != "#E01E5A" {
		t.Fatalf("color=%v", att["color"])
	}
	blocks := att["blocks"].([]any)
	if len(blocks) < 3 {
		t.Fatalf("blocks=%d", len(blocks))
	}
}

func TestRenderAlertEmail(t *testing.T) {
	a := &Alerter{}
	html := a.renderAlertEmail(AlertMeta{
		Event:        "DOWN",
		Name:         "Google",
		URL:          "https://www.google.com/",
		Message:      "connection refused",
		DashboardURL: "http://localhost/monitors/1",
		ResponseMs:   176,
		IncidentID:   "deadbeef",
		EventAt:      time.Now().UTC(),
	})
	for _, want := range []string{"MONITOR DOWN", "Google", "https://www.google.com/", "INC-DEADBEEF", "Open in Sentinel", "#E01E5A"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in email html", want)
		}
	}
}
