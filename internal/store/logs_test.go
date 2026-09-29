package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func TestLogsDBBatchAndPrune(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "sentinel.db")
	st, err := Open(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	logs, err := OpenLogsDB(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logs.Close()

	now := time.Now().UTC()
	err = logs.InsertEventsBatch([]models.LogEvent{
		{HostID: "h1", SourceID: "s1", Timestamp: now, Level: models.LogLevelError, Message: "boom", Fingerprint: "a", ReceivedAt: now},
		{HostID: "h1", SourceID: "s1", Timestamp: now.Add(-48 * time.Hour), Level: models.LogLevelWarn, Message: "old", Fingerprint: "b", ReceivedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := logs.QueryEvents(LogEventQuery{HostID: "h1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 2 {
		t.Fatalf("want 2 events, got %d", len(ev))
	}
	n, err := logs.PruneEvents(now.Add(-24*time.Hour), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want pruned 1, got %d", n)
	}
}

func TestLogSourceCRUD(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Need a host for FK
	h := &models.Host{Name: "web", TenantID: "", Enabled: true, Status: models.HostPending}
	models.ApplyHostDefaults(h)
	if err := st.CreateHost(h); err != nil {
		t.Fatal(err)
	}
	src := &models.LogSource{
		HostID: h.ID, Name: "nginx", Type: models.LogSourceFile,
		Path: "/var/log/nginx/error.log", Enabled: true, MinLevel: models.LogLevelWarn,
		IncludePatterns: []string{"ERROR"},
	}
	if err := st.CreateLogSource(src); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListLogSources(h.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Path != src.Path {
		t.Fatalf("unexpected list: %+v", list)
	}
	settings, err := st.GetLogSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.RetentionDays != 7 {
		t.Fatalf("default retention: %d", settings.RetentionDays)
	}
}
