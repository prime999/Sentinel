CREATE TABLE IF NOT EXISTS monitors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'GET',
    expected_status INTEGER NOT NULL DEFAULT 200,
    expected_status_min INTEGER,
    expected_status_max INTEGER,
    keyword_must_exist TEXT,
    keyword_must_not_exist TEXT,
    request_body TEXT,
    request_headers TEXT,
    interval_seconds INTEGER NOT NULL DEFAULT 60,
    timeout_ms INTEGER NOT NULL DEFAULT 10000,
    slow_threshold_ms INTEGER NOT NULL DEFAULT 3000,
    follow_redirects INTEGER NOT NULL DEFAULT 1,
    alert_emails TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_status TEXT NOT NULL DEFAULT 'unknown',
    last_checked_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS check_results (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER NOT NULL DEFAULT 0,
    dns_ms INTEGER,
    tcp_ms INTEGER,
    tls_ms INTEGER,
    ttfb_ms INTEGER,
    error TEXT,
    checked_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_check_results_monitor_checked
    ON check_results(monitor_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS incidents (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    message TEXT,
    started_at TEXT NOT NULL,
    resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_incidents_monitor
    ON incidents(monitor_id, started_at DESC);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    expires_at TEXT NOT NULL
);
