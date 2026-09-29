package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func testMonitor(id, name string) *models.Monitor {
	return &models.Monitor{
		ID:              id,
		Type:            models.MonitorHeartbeat,
		Name:            name,
		URL:             "",
		Method:          "GET",
		IntervalSeconds: 60,
		TimeoutMs:       10000,
		SlowThresholdMs: 3000,
		ExpectedStatus:  200,
		Enabled:         true,
		HeartbeatToken:  "hb-test-token-" + id[:8],
		Tags:            []string{},
	}
}

func TestMonitorBackupExportAndImport(t *testing.T) {
	srv, st, admin := newTestMFAServer(t)
	h := srv.Handler()
	adminSess := withSession(t, st, admin.ID)

	m1 := testMonitor("11111111-1111-1111-1111-111111111111", "Site A")
	if err := st.InsertMonitorImport(m1); err != nil {
		t.Fatal(err)
	}

	export := authedReq(t, h, http.MethodGet, "/api/settings/monitor-backup", adminSess)
	if export.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", export.Code, export.Body.String())
	}
	var file platformBackupFile
	if err := json.Unmarshal(export.Body.Bytes(), &file); err != nil {
		t.Fatal(err)
	}
	if file.Format != platformBackupFormat || len(file.Monitors) != 1 {
		t.Fatalf("file=%+v", file)
	}
	if file.Monitors[0].HeartbeatToken != m1.HeartbeatToken {
		t.Fatalf("expected heartbeat token in export")
	}

	backupRow := file.Monitors[0]
	backupRow.Name = "Site A restored"
	preview := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/preview", adminSess, file)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	var prev platformBackupPreview
	if err := json.Unmarshal(preview.Body.Bytes(), &prev); err != nil {
		t.Fatal(err)
	}
	if prev.Monitors.Total != 1 || len(prev.Monitors.Conflicts) != 1 {
		t.Fatalf("preview=%+v", prev)
	}

	importCreate := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/import", adminSess, map[string]any{
		"mode":     "create_only",
		"format":   file.Format,
		"monitors": []models.Monitor{backupRow},
	})
	if importCreate.Code != http.StatusOK {
		t.Fatalf("import create_only status=%d body=%s", importCreate.Code, importCreate.Body.String())
	}
	var res platformBackupImportResult
	if err := json.Unmarshal(importCreate.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Monitors.Created != 0 || res.Monitors.Updated != 0 || res.Monitors.Skipped != 1 {
		t.Fatalf("create_only result=%+v", res)
	}

	importOverwrite := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/import", adminSess, map[string]any{
		"mode":     "overwrite",
		"format":   file.Format,
		"monitors": []models.Monitor{backupRow},
	})
	if importOverwrite.Code != http.StatusOK {
		t.Fatalf("import overwrite status=%d body=%s", importOverwrite.Code, importOverwrite.Body.String())
	}
	if err := json.Unmarshal(importOverwrite.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Monitors.Updated != 1 {
		t.Fatalf("overwrite result=%+v", res)
	}
	stored, err := st.GetMonitor(m1.ID)
	if err != nil || stored == nil || stored.Name != "Site A restored" || stored.HeartbeatToken != m1.HeartbeatToken {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestLegacyMonitorBackupImport(t *testing.T) {
	srv, st, admin := newTestMFAServer(t)
	h := srv.Handler()
	adminSess := withSession(t, st, admin.ID)

	m1 := testMonitor("11111111-1111-1111-1111-111111111111", "Legacy")
	legacy := monitorBackupFile{
		Format:   monitorBackupFormat,
		Version:  1,
		Monitors: []models.Monitor{*m1},
	}
 imp := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/import", adminSess, map[string]any{
		"mode":     "create_only",
		"format":   legacy.Format,
		"monitors": legacy.Monitors,
	})
	if imp.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", imp.Code, imp.Body.String())
	}
	var res platformBackupImportResult
	if err := json.Unmarshal(imp.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Monitors.Created != 1 {
		t.Fatalf("result=%+v", res)
	}
}

func TestPlatformBackupCustomerAndMonitor(t *testing.T) {
	srv, st, admin := newTestMFAServer(t)
	h := srv.Handler()
	adminSess := withSession(t, st, admin.ID)

	custID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	monID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	file := platformBackupFile{
		Format: platformBackupFormat,
		Customers: []models.Customer{{
			ID:           custID,
			Name:         "Restored Co",
			MonitorQuota: 10,
		}},
		Monitors: []models.Monitor{{
			ID:              monID,
			Type:            models.MonitorHeartbeat,
			Name:            "Svc",
			IntervalSeconds: 60,
			ExpectedStatus:  200,
			Enabled:         true,
			TenantID:        custID,
			HeartbeatToken:  "hb-restore",
			Tags:            []string{},
		}},
	}
 imp := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/import", adminSess, map[string]any{
		"mode":      "create_only",
		"format":    file.Format,
		"customers": file.Customers,
		"monitors":  file.Monitors,
	})
	if imp.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", imp.Code, imp.Body.String())
	}
	var res platformBackupImportResult
	if err := json.Unmarshal(imp.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Customers.Created != 1 || res.Monitors.Created != 1 {
		t.Fatalf("result=%+v", res)
	}
	c, _ := st.GetCustomer(custID)
	if c == nil || c.Name != "Restored Co" {
		t.Fatalf("customer=%+v", c)
	}
	m, _ := st.GetMonitor(monID)
	if m == nil || m.TenantID != custID {
		t.Fatalf("monitor=%+v", m)
	}
	_ = admin
}

func TestMonitorBackupAuth(t *testing.T) {
	srv, st, admin := newTestMFAServer(t)
	h := srv.Handler()

	rec := authedReq(t, h, http.MethodGet, "/api/settings/monitor-backup", "invalid")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid session status=%d", rec.Code)
	}

	viewer, err := st.CreateUser("viewer1", "v@example.com", "Viewer1234", models.RoleViewer, "")
	if err != nil {
		t.Fatal(err)
	}
	viewerSess := withSession(t, st, viewer.ID)
	forbidden := authedReq(t, h, http.MethodGet, "/api/settings/monitor-backup", viewerSess)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("viewer export status=%d body=%s", forbidden.Code, forbidden.Body.String())
	}

	adminSess := withSession(t, st, admin.ID)
	ok := authedReq(t, h, http.MethodGet, "/api/settings/monitor-backup", adminSess)
	if ok.Code != http.StatusOK {
		t.Fatalf("admin export status=%d body=%s", ok.Code, ok.Body.String())
	}
}

func TestMonitorBackupCustomerAdminTenantScope(t *testing.T) {
	srv, st, _ := newTestMFAServer(t)
	h := srv.Handler()

	cust, err := st.CreateCustomer("Acme", 5)
	if err != nil {
		t.Fatal(err)
	}
	tenantAdmin, err := st.CreateUser("custadmin", "c@example.com", "Admin1234", models.RoleAdmin, cust.ID)
	if err != nil {
		t.Fatal(err)
	}
	tenantSess := withSession(t, st, tenantAdmin.ID)

	importBody := map[string]any{
		"mode": "create_only",
		"format": platformBackupFormat,
		"monitors": []models.Monitor{{
			ID:              "33333333-3333-3333-3333-333333333333",
			Type:            models.MonitorHeartbeat,
			Name:            "Imported",
			URL:             "",
			Method:          "GET",
			IntervalSeconds: 60,
			ExpectedStatus:  200,
			HeartbeatToken:  "imported-hb-token",
			TenantID:        "wrong-tenant",
			Tags:            []string{},
		}},
	}
 imp := authedJSON(t, h, http.MethodPost, "/api/settings/monitor-backup/import", tenantSess, importBody)
	if imp.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", imp.Code, imp.Body.String())
	}
	var res platformBackupImportResult
	if err := json.Unmarshal(imp.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Monitors.Created != 1 {
		t.Fatalf("result=%+v", res)
	}
	stored, err := st.GetMonitor("33333333-3333-3333-3333-333333333333")
	if err != nil || stored == nil || stored.TenantID != cust.ID {
		t.Fatalf("tenant forced: %+v err=%v", stored, err)
	}
}
