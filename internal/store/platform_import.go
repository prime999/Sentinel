package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/config"
	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func (s *Store) ListCustomersForExport(tenantID string) ([]models.Customer, error) {
	if tenantID != "" {
		c, err := s.GetCustomer(tenantID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return []models.Customer{}, nil
		}
		return []models.Customer{*c}, nil
	}
	return s.ListCustomers()
}

func (s *Store) GetCustomersByIDs(ids []string) (map[string]models.Customer, error) {
	out := make(map[string]models.Customer, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`SELECT id, name, monitor_quota, alert_emails, created_at FROM customers WHERE id IN (`+sqlPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanCustomer(rows, false)
		if err != nil {
			return nil, err
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

func (s *Store) InsertCustomerImport(c *models.Customer) error {
	if strings.TrimSpace(c.ID) == "" {
		c.ID = newID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.MonitorQuota < 1 {
		c.MonitorQuota = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO customers (id, name, monitor_quota, alert_emails, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.MonitorQuota, strings.TrimSpace(c.AlertEmails), formatTime(c.CreatedAt),
	)
	return err
}

func (s *Store) ReplaceCustomerImport(c *models.Customer) error {
	if c.MonitorQuota < 1 {
		c.MonitorQuota = 1
	}
	res, err := s.db.Exec(`
		UPDATE customers SET name = ?, monitor_quota = ?, alert_emails = ? WHERE id = ?`,
		c.Name, c.MonitorQuota, strings.TrimSpace(c.AlertEmails), c.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (s *Store) ListUsersForExport(tenantID string) ([]models.User, error) {
	if tenantID != "" {
		return s.ListUsersByTenant(tenantID)
	}
	return s.ListUsers()
}

func (s *Store) GetUsersByIDs(ids []string) (map[string]models.User, error) {
	out := make(map[string]models.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`
		SELECT id, username, name, email, password_hash, mfa_enabled, role, tenant_id, created_at, updated_at
		FROM users WHERE id IN (`+sqlPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out[u.ID] = u
	}
	return out, rows.Err()
}

func (s *Store) InsertUserImport(u *models.User) error {
	if strings.TrimSpace(u.ID) == "" {
		u.ID = newID()
	}
	if strings.TrimSpace(u.Username) == "" {
		return fmt.Errorf("username required")
	}
	if strings.TrimSpace(u.PasswordHash) == "" {
		return fmt.Errorf("password_hash required")
	}
	if u.Role != models.RoleAdmin && u.Role != models.RoleViewer {
		u.Role = models.RoleViewer
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, name, email, password_hash, mfa_enabled, role, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.Name, u.Email, u.PasswordHash, boolToInt(u.MFAEnabled), u.Role,
		nullString(u.TenantID), formatTime(u.CreatedAt), formatTime(u.UpdatedAt),
	)
	return err
}

func (s *Store) ReplaceUserImport(u *models.User) error {
	if strings.TrimSpace(u.PasswordHash) == "" {
		return fmt.Errorf("password_hash required")
	}
	if u.Role != models.RoleAdmin && u.Role != models.RoleViewer {
		u.Role = models.RoleViewer
	}
	res, err := s.db.Exec(`
		UPDATE users SET username = ?, name = ?, email = ?, password_hash = ?, mfa_enabled = ?, role = ?, tenant_id = ?, updated_at = ?
		WHERE id = ?`,
		u.Username, u.Name, u.Email, u.PasswordHash, boolToInt(u.MFAEnabled), u.Role,
		nullString(u.TenantID), formatTime(time.Now().UTC()), u.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *Store) ListPerformanceTargetsForExport(tenantID string) ([]models.PerformanceTarget, error) {
	q := `SELECT ` + perfTargetColumns + ` FROM performance_targets`
	var args []any
	if tenantID != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, tenantID)
	}
	q += ` ORDER BY created_at ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PerformanceTarget
	for rows.Next() {
		t, _, err := scanPerformanceTargetRow(rows, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) GetPerformanceTargetsByIDs(ids []string) (map[string]models.PerformanceTarget, error) {
	out := make(map[string]models.PerformanceTarget, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`SELECT `+perfTargetColumns+` FROM performance_targets WHERE id IN (`+sqlPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		t, _, err := scanPerformanceTargetRow(rows, false)
		if err != nil {
			return nil, err
		}
		out[t.ID] = *t
	}
	return out, rows.Err()
}

func preparePerformanceTargetImport(t *models.PerformanceTarget) {
	now := time.Now().UTC()
	if strings.TrimSpace(t.ID) == "" {
		t.ID = newID()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	if t.Method == "" {
		t.Method = "GET"
	}
	if t.IntervalSeconds < 30 {
		if t.IntervalSeconds == 0 {
			t.IntervalSeconds = 300
		} else {
			t.IntervalSeconds = 30
		}
	}
	if t.TimeoutMs == 0 {
		t.TimeoutMs = 10000
	}
	if t.SlowThresholdMs == 0 {
		t.SlowThresholdMs = 3000
	}
	if t.AlertAfterSlow < 1 {
		t.AlertAfterSlow = 2
	}
	t.ConsecutiveSlow = 0
	t.LastStatus = models.StatusUnknown
	t.LastCheckedAt = nil
}

func (s *Store) InsertPerformanceTargetImport(t *models.PerformanceTarget) error {
	preparePerformanceTargetImport(t)
	_, err := s.db.Exec(`
		INSERT INTO performance_targets (`+perfTargetColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.URL, t.Method, t.IntervalSeconds, t.TimeoutMs, t.SlowThresholdMs,
		boolToInt(t.FollowRedirects), boolToInt(t.Enabled), t.AlertEmails,
		t.HTTPUsername, t.HTTPPassword, nullString(t.TenantID),
		t.AlertAfterSlow, t.ConsecutiveSlow,
		string(t.LastStatus), nil, formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	)
	return err
}

func (s *Store) ReplacePerformanceTargetImport(t *models.PerformanceTarget) error {
	preparePerformanceTargetImport(t)
	res, err := s.db.Exec(`
		UPDATE performance_targets SET
			name=?, url=?, method=?, interval_seconds=?, timeout_ms=?, slow_threshold_ms=?,
			follow_redirects=?, enabled=?, alert_emails=?, http_username=?, http_password=?,
			tenant_id=?, alert_after_slow=?, consecutive_slow=?, last_status=?, last_checked_at=?, updated_at=?
		WHERE id=?`,
		t.Name, t.URL, t.Method, t.IntervalSeconds, t.TimeoutMs, t.SlowThresholdMs,
		boolToInt(t.FollowRedirects), boolToInt(t.Enabled), t.AlertEmails,
		t.HTTPUsername, t.HTTPPassword, nullString(t.TenantID),
		t.AlertAfterSlow, t.ConsecutiveSlow, string(t.LastStatus), nil, formatTime(t.UpdatedAt), t.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("performance target not found")
	}
	return nil
}

func (s *Store) ListHostsForExport(tenantID string) ([]models.HostBackup, error) {
	var hosts []models.Host
	var err error
	if tenantID != "" {
		hosts, err = s.ListHostsByTenant(tenantID)
	} else {
		hosts, err = s.ListHosts()
	}
	if err != nil {
		return nil, err
	}
	out := make([]models.HostBackup, len(hosts))
	for i, h := range hosts {
		h.LastSeenAt = nil
		h.Status = models.HostPending
		h.ConsecutiveMisses = 0
		h.Disks = nil
		h.ServiceStatus = nil
		h.Security = nil
		out[i] = models.HostBackup{Host: h}
	}
	if len(hosts) == 0 {
		return out, nil
	}
	ids := make([]any, len(hosts))
	for i, h := range hosts {
		ids[i] = h.ID
	}
	rows, err := s.db.Query(`SELECT id, token_hash FROM hosts WHERE id IN (`+sqlPlaceholders(len(hosts))+`)`, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hashes := map[string]string{}
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, err
		}
		hashes[id] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].TokenHash = hashes[out[i].ID]
	}
	return out, nil
}

func (s *Store) GetHostsByIDs(ids []string) (map[string]models.Host, error) {
	out := make(map[string]models.Host, len(ids))
	for _, id := range ids {
		h, err := s.GetHost(id)
		if err != nil {
			return nil, err
		}
		if h != nil {
			out[id] = *h
		}
	}
	return out, nil
}

func prepareHostImport(h *models.Host) {
	models.ApplyHostDefaults(h)
	now := time.Now().UTC()
	if strings.TrimSpace(h.ID) == "" {
		h.ID = newID()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = now
	}
	h.UpdatedAt = now
	h.LastSeenAt = nil
	h.ConsecutiveMisses = 0
	if h.Status == "" {
		h.Status = models.HostPending
	}
	h.Disks = nil
	h.ServiceStatus = nil
}

func (s *Store) insertHostRow(h *models.Host, tokenHash string) error {
	prepareHostImport(h)
	if tokenHash == "" {
		tokenHash = ""
	}
	_, err := s.db.Exec(`
		INSERT INTO hosts (
			id, name, hostname, tenant_id, os, os_version, kernel_version, arch, agent_version, token_hash,
			num_cpu, reboot_required, last_seen_at, status,
			enabled, interval_seconds, alert_after_failures, consecutive_misses,
			collect_cpu, collect_memory, collect_disk, collect_load, collect_swap, collect_iowait, collect_security, collect_services,
			alert_cpu_enabled, alert_cpu_warning, alert_cpu_threshold, alert_cpu_after,
			alert_memory_enabled, alert_memory_warning, alert_memory_threshold, alert_memory_after,
			alert_disk_enabled, alert_disk_warning, alert_disk_threshold, alert_disk_after,
			alert_load_enabled, alert_load_warning, alert_load_threshold, alert_load_after,
			alert_swap_enabled, alert_swap_warning, alert_swap_threshold, alert_swap_after,
			alert_iowait_enabled, alert_iowait_warning, alert_iowait_threshold, alert_iowait_after,
			alert_auth_enabled, alert_auth_threshold, alert_root_login_enabled, alert_reboot_enabled, alert_service_enabled,
			services_json, security_json, disks_latest_json, services_latest_json,
			alert_emails, notify_email, notify_slack, notify_webhooks,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		h.ID, h.Name, h.Hostname, h.TenantID, h.OS, h.OSVersion, h.KernelVersion, h.Arch, h.AgentVersion, tokenHash,
		h.NumCPU, boolToInt(h.RebootRequired), nil, string(h.Status),
		boolToInt(h.Enabled), h.IntervalSeconds, h.AlertAfterFailures, h.ConsecutiveMisses,
		boolToInt(h.CollectCPU), boolToInt(h.CollectMemory), boolToInt(h.CollectDisk), boolToInt(h.CollectLoad),
		boolToInt(h.CollectSwap), boolToInt(h.CollectIOWait), boolToInt(h.CollectSecurity), boolToInt(h.CollectServices),
		boolToInt(h.AlertCPUEnabled), h.AlertCPUWarning, h.AlertCPUThreshold, h.AlertCPUAfter,
		boolToInt(h.AlertMemoryEnabled), h.AlertMemoryWarning, h.AlertMemoryThreshold, h.AlertMemoryAfter,
		boolToInt(h.AlertDiskEnabled), h.AlertDiskWarning, h.AlertDiskThreshold, h.AlertDiskAfter,
		boolToInt(h.AlertLoadEnabled), h.AlertLoadWarning, h.AlertLoadThreshold, h.AlertLoadAfter,
		boolToInt(h.AlertSwapEnabled), h.AlertSwapWarning, h.AlertSwapThreshold, h.AlertSwapAfter,
		boolToInt(h.AlertIOWaitEnabled), h.AlertIOWaitWarning, h.AlertIOWaitThreshold, h.AlertIOWaitAfter,
		boolToInt(h.AlertAuthEnabled), h.AlertAuthThreshold, boolToInt(h.AlertRootLoginEnabled), boolToInt(h.AlertRebootEnabled), boolToInt(h.AlertServiceEnabled),
		marshalJSON(h.Services), "{}", "[]", "[]",
		h.AlertEmails, boolToInt(h.NotifyEmail), boolToInt(h.NotifySlack), boolToInt(h.NotifyWebhooks),
		formatTime(h.CreatedAt), formatTime(h.UpdatedAt),
	)
	return err
}

func hostTokenHashFromBackup(b models.HostBackup) string {
	if strings.TrimSpace(b.IngestToken) != "" {
		return hashAPIToken(b.IngestToken)
	}
	return strings.TrimSpace(b.TokenHash)
}

func (s *Store) InsertHostImport(b models.HostBackup) error {
	return s.insertHostRow(&b.Host, hostTokenHashFromBackup(b))
}

func (s *Store) ReplaceHostImport(b models.HostBackup) error {
	prepareHostImport(&b.Host)
	tokenHash := hostTokenHashFromBackup(b)
	res, err := s.db.Exec(`
		UPDATE hosts SET
			name = ?, hostname = ?, tenant_id = ?, os = ?, os_version = ?, kernel_version = ?, arch = ?, agent_version = ?,
			token_hash = CASE WHEN ? != '' THEN ? ELSE token_hash END,
			num_cpu = ?, reboot_required = ?, last_seen_at = ?, status = ?, enabled = ?, interval_seconds = ?,
			alert_after_failures = ?, consecutive_misses = ?,
			collect_cpu = ?, collect_memory = ?, collect_disk = ?, collect_load = ?, collect_swap = ?, collect_iowait = ?, collect_security = ?, collect_services = ?,
			alert_cpu_enabled = ?, alert_cpu_warning = ?, alert_cpu_threshold = ?, alert_cpu_after = ?,
			alert_memory_enabled = ?, alert_memory_warning = ?, alert_memory_threshold = ?, alert_memory_after = ?,
			alert_disk_enabled = ?, alert_disk_warning = ?, alert_disk_threshold = ?, alert_disk_after = ?,
			alert_load_enabled = ?, alert_load_warning = ?, alert_load_threshold = ?, alert_load_after = ?,
			alert_swap_enabled = ?, alert_swap_warning = ?, alert_swap_threshold = ?, alert_swap_after = ?,
			alert_iowait_enabled = ?, alert_iowait_warning = ?, alert_iowait_threshold = ?, alert_iowait_after = ?,
			alert_auth_enabled = ?, alert_auth_threshold = ?, alert_root_login_enabled = ?, alert_reboot_enabled = ?, alert_service_enabled = ?,
			services_json = ?, alert_emails = ?, notify_email = ?, notify_slack = ?, notify_webhooks = ?,
			updated_at = ?
		WHERE id = ?`,
		b.Name, b.Hostname, b.TenantID, b.OS, b.OSVersion, b.KernelVersion, b.Arch, b.AgentVersion,
		tokenHash, tokenHash,
		b.NumCPU, boolToInt(b.RebootRequired), nil, string(b.Status), boolToInt(b.Enabled), b.IntervalSeconds,
		b.AlertAfterFailures, b.ConsecutiveMisses,
		boolToInt(b.CollectCPU), boolToInt(b.CollectMemory), boolToInt(b.CollectDisk), boolToInt(b.CollectLoad),
		boolToInt(b.CollectSwap), boolToInt(b.CollectIOWait), boolToInt(b.CollectSecurity), boolToInt(b.CollectServices),
		boolToInt(b.AlertCPUEnabled), b.AlertCPUWarning, b.AlertCPUThreshold, b.AlertCPUAfter,
		boolToInt(b.AlertMemoryEnabled), b.AlertMemoryWarning, b.AlertMemoryThreshold, b.AlertMemoryAfter,
		boolToInt(b.AlertDiskEnabled), b.AlertDiskWarning, b.AlertDiskThreshold, b.AlertDiskAfter,
		boolToInt(b.AlertLoadEnabled), b.AlertLoadWarning, b.AlertLoadThreshold, b.AlertLoadAfter,
		boolToInt(b.AlertSwapEnabled), b.AlertSwapWarning, b.AlertSwapThreshold, b.AlertSwapAfter,
		boolToInt(b.AlertIOWaitEnabled), b.AlertIOWaitWarning, b.AlertIOWaitThreshold, b.AlertIOWaitAfter,
		boolToInt(b.AlertAuthEnabled), b.AlertAuthThreshold, boolToInt(b.AlertRootLoginEnabled), boolToInt(b.AlertRebootEnabled), boolToInt(b.AlertServiceEnabled),
		marshalJSON(b.Services), b.AlertEmails, boolToInt(b.NotifyEmail), boolToInt(b.NotifySlack), boolToInt(b.NotifyWebhooks),
		formatTime(time.Now().UTC()), b.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("host not found")
	}
	return nil
}

func (s *Store) ListMaintenanceWindowsForExport(tenantID string, monitorIDs []string) ([]models.MaintenanceWindow, error) {
	all, err := s.ListMaintenanceWindows()
	if err != nil {
		return nil, err
	}
	if tenantID == "" {
		return all, nil
	}
	allowed := map[string]bool{"": true}
	for _, id := range monitorIDs {
		allowed[id] = true
	}
	var out []models.MaintenanceWindow
	for _, w := range all {
		if w.MonitorID == "" || allowed[w.MonitorID] {
			out = append(out, w)
		}
	}
	return out, nil
}

func (s *Store) GetMaintenanceWindowsByIDs(ids []string) (map[string]models.MaintenanceWindow, error) {
	out := make(map[string]models.MaintenanceWindow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.Query(`
		SELECT id, name, monitor_id, starts_at, ends_at, created_at
		FROM maintenance_windows WHERE id IN (`+sqlPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		w, err := scanMaintenance(rows)
		if err != nil {
			return nil, err
		}
		out[w.ID] = *w
	}
	return out, rows.Err()
}

func (s *Store) InsertMaintenanceImport(w *models.MaintenanceWindow) error {
	if strings.TrimSpace(w.ID) == "" {
		w.ID = newID()
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now().UTC()
	}
	var monitorID interface{}
	if w.MonitorID != "" {
		monitorID = w.MonitorID
	}
	_, err := s.db.Exec(`
		INSERT INTO maintenance_windows (id, name, monitor_id, starts_at, ends_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, monitorID, formatTime(w.StartsAt), formatTime(w.EndsAt), formatTime(w.CreatedAt),
	)
	return err
}

func (s *Store) ReplaceMaintenanceImport(w *models.MaintenanceWindow) error {
	var monitorID interface{}
	if w.MonitorID != "" {
		monitorID = w.MonitorID
	}
	res, err := s.db.Exec(`
		UPDATE maintenance_windows SET name = ?, monitor_id = ?, starts_at = ?, ends_at = ? WHERE id = ?`,
		w.Name, monitorID, formatTime(w.StartsAt), formatTime(w.EndsAt), w.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("maintenance window not found")
	}
	return nil
}

func (s *Store) ListSettingKeysLike(prefix string) ([]string, error) {
	rows, err := s.db.Query(`SELECT key FROM settings WHERE key LIKE ?`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func parseSlackTenantFromKey(key string) string {
	if key == "slack" {
		return ""
	}
	if strings.HasPrefix(key, "slack:") {
		return strings.TrimPrefix(key, "slack:")
	}
	return ""
}

func (s *Store) ExportPlatformSettings(includePlatform bool, tenantID string, smtpFallback models.SMTPConfig, serverFallback config.ServerConfig) (models.PlatformSettingsBackup, error) {
	var out models.PlatformSettingsBackup
	if includePlatform {
		org, err := s.GetOrgSettings()
		if err != nil {
			return out, err
		}
		out.Org = &org
		smtp, err := s.GetSMTPConfig(smtpFallback)
		if err != nil {
			return out, err
		}
		out.SMTP = &smtp
		wh, err := s.GetWebhooks()
		if err != nil {
			return out, err
		}
		out.Webhooks = wh
		srv, err := s.GetServerSettings(serverFallback)
		if err != nil {
			return out, err
		}
		out.Server = &srv
		sp, err := s.GetStatusPageConfig()
		if err != nil {
			return out, err
		}
		out.StatusPage = &sp
		rows, err := s.db.Query(`SELECT key FROM settings WHERE key = 'slack' OR key LIKE 'slack:%'`)
		if err != nil {
			return out, err
		}
		defer rows.Close()
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				return out, err
			}
			tid := parseSlackTenantFromKey(key)
			cfg, err := s.GetSlackConfig(tid)
			if err != nil {
				return out, err
			}
			out.Slack = append(out.Slack, models.SlackSettingBackup{TenantID: tid, Config: cfg})
		}
		if err := rows.Err(); err != nil {
			return out, err
		}
	} else if tenantID != "" {
		cfg, err := s.GetSlackConfig(tenantID)
		if err != nil {
			return out, err
		}
		out.Slack = []models.SlackSettingBackup{{TenantID: tenantID, Config: cfg}}
	}
	return out, nil
}

func (s *Store) ImportPlatformSettings(settings *models.PlatformSettingsBackup, overwrite bool) error {
	if settings == nil {
		return nil
	}
	if settings.Org != nil {
		if err := s.SaveOrgSettings(*settings.Org); err != nil {
			return err
		}
	}
	if settings.SMTP != nil {
		if err := s.SaveSMTPConfig(*settings.SMTP); err != nil {
			return err
		}
	}
	if settings.Webhooks != nil {
		if err := s.SaveWebhooks(settings.Webhooks); err != nil {
			return err
		}
	}
	if settings.Server != nil {
		if err := s.SaveServerSettings(*settings.Server); err != nil {
			return err
		}
	}
	if settings.Logs != nil {
		if err := s.SaveLogSettings(*settings.Logs); err != nil {
			return err
		}
	}
	if settings.StatusPage != nil {
		if err := s.SaveStatusPageConfig(*settings.StatusPage); err != nil {
			return err
		}
	}
	for _, sl := range settings.Slack {
		if err := s.SaveSlackConfig(sl.TenantID, sl.Config); err != nil {
			return err
		}
	}
	_ = overwrite
	return nil
}

func (s *Store) CustomerExists(id string) (bool, error) {
	if id == "" {
		return true, nil
	}
	c, err := s.GetCustomer(id)
	if err != nil {
		return false, err
	}
	return c != nil, nil
}

func (s *Store) MonitorExists(id string) (bool, error) {
	if id == "" {
		return true, nil
	}
	m, err := s.GetMonitor(id)
	if err != nil {
		return false, err
	}
	return m != nil, nil
}

