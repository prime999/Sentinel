export type MonitorType = 'http' | 'port' | 'ssl' | 'dns' | 'heartbeat'

export interface Monitor {
  id: string
  type: MonitorType
  name: string
  url: string
  port?: number
  config?: string
  method: string
  expected_status: number
  expected_status_min?: number
  expected_status_max?: number
  keyword_must_exist: string
  keyword_must_not_exist: string
  request_body: string
  request_headers: string
  interval_seconds: number
  timeout_ms: number
  slow_threshold_ms: number
  follow_redirects: boolean
  alert_emails: string
  enabled: boolean
  invert?: boolean
  tags?: string[]
  heartbeat_token?: string
  tenant_id?: string
  alert_after_failures?: number
  consecutive_failures?: number
  last_status: 'up' | 'down' | 'degraded' | 'unknown'
  last_checked_at?: string
  latest_response_time_ms?: number
}

export interface CheckResult {
  id: string
  monitor_id: string
  status: string
  status_code?: number
  response_time_ms: number
  dns_ms?: number
  tcp_ms?: number
  tls_ms?: number
  ttfb_ms?: number
  error?: string
  details?: string
  checked_at: string
}

export interface PaginatedResults<T> {
  items: T[]
  total: number
  limit: number
  offset: number
}

export interface SSLDetails {
  expires_at: string
  days_remaining: number
  issuer: string
  subject: string
  fingerprint: string
  issues?: string[]
}

export interface DNSDetails {
  records: Record<string, string[]>
  changes?: { type: string; before: string; after: string }[]
}

export interface PortDetails {
  host: string
  port: number
  open: boolean
}

export interface PerformanceMetrics {
  avg_ms: number
  min_ms: number
  max_ms: number
  p50_ms: number
  p95_ms: number
  p99_ms: number
  slow_count: number
  degraded_pct: number
}

export interface MonitorStats {
  monitor_id: string
  points: {
    timestamp: string
    response_time_ms: number
    status: string
    dns_ms?: number
    tcp_ms?: number
    tls_ms?: number
    ttfb_ms?: number
  }[]
  uptime_pct: number
  avg_response_ms: number
  performance: PerformanceMetrics
}

export type PerformanceHealth = 'good' | 'warning' | 'critical' | 'collecting'

export interface ServicePerformance {
  service_id: string
  name: string
  probe_type: string
  target: string
  latency_status: string
  slow_threshold_ms: number
  avg_ms: number
  p95_ms: number
  max_ms: number
  slow_count: number
  check_count: number
  has_data: boolean
  health: PerformanceHealth
}

export interface FleetTimelinePoint {
  timestamp: string
  avg_ms: number
  check_count: number
  slow_count: number
}

export interface PerformanceTarget {
  id: string
  name: string
  url: string
  method: string
  interval_seconds: number
  timeout_ms: number
  slow_threshold_ms: number
  follow_redirects: boolean
  enabled: boolean
  alert_emails?: string
  tenant_id?: string
  alert_after_slow?: number
  consecutive_slow?: number
  last_status: 'up' | 'down' | 'degraded' | 'unknown'
  last_checked_at?: string
  latest_response_time_ms?: number
}

export interface PerformanceResult {
  id: string
  target_id: string
  status: string
  status_code?: number
  response_time_ms: number
  dns_ms?: number
  tcp_ms?: number
  tls_ms?: number
  ttfb_ms?: number
  error?: string
  checked_at: string
}

export interface PerformanceStats {
  target_id: string
  points: MonitorStats['points']
  avg_response_ms: number
  performance: PerformanceMetrics
}

export interface FleetPerformance {
  period: string
  timeline: FleetTimelinePoint[]
  services: ServicePerformance[]
  total_checks: number
  avg_ms: number
  p95_ms: number
  slow_checks: number
  service_count: number
  healthy_count: number
  warning_count: number
  critical_count: number
  collecting_count: number
}

export type UserRole = 'admin' | 'viewer'

export interface Profile {
  id: string
  username: string
  name: string
  email: string
  role: UserRole
  tenant_id?: string
}

export interface Customer {
  id: string
  name: string
  monitor_quota: number
  monitor_count?: number
  created_at: string
}

export interface TeamMember {
  id: string
  username: string
  email: string
  role: UserRole
  tenant_id?: string
  created_at: string
}

export interface CreateTeamMemberRequest {
  username: string
  email?: string
  password: string
  role: UserRole
  tenant_id?: string
}

export interface UpdateTeamMemberRequest {
  username?: string
  email?: string
  password?: string
  role?: UserRole
  tenant_id?: string
}

export interface UpdateProfileRequest {
  current_password: string
  username?: string
  name?: string
  email?: string
  new_password?: string
}

export interface OrgSettings {
  company_name: string
  tagline: string
  logo: string
}

export interface SMTPConfig {
  host: string
  port: number
  username: string
  password: string
  from: string
  alert_emails: string
  tls: boolean
}

export interface WebhookConfig {
  url: string
  enabled: boolean
  events: string[]
}

export interface MaintenanceWindow {
  id: string
  name: string
  monitor_id?: string
  starts_at: string
  ends_at: string
  created_at: string
}

export interface ServerSettings {
  dashboard_url: string
  retention_days: number
  workers: number
}

export interface StatusPageConfig {
  enabled: boolean
  title: string
  monitor_ids: string[]
}

export interface PublicMonitorStatus {
  id: string
  name: string
  type: string
  status: string
  last_checked_at?: string
  url?: string
}

export interface PublicStatusResponse {
  title: string
  monitors: PublicMonitorStatus[]
}

export interface Incident {
  id: string
  monitor_id: string
  monitor_name: string
  type: string
  message?: string
  started_at: string
  resolved_at?: string
}

export interface AuditEntry {
  id: string
  actor: string
  action: string
  resource: string
  detail?: string
  created_at: string
}

export interface APIToken {
  id: string
  user_id: string
  name: string
  prefix: string
  created_at: string
  last_used_at?: string
}

export interface APITokenCreated extends APIToken {
  token: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  login: (username: string, password: string) =>
    request<{ ok: boolean }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  forgotPassword: (email: string) =>
    request<{ message: string }>('/api/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),
  resetPassword: (token: string, new_password: string) =>
    request<{ ok: boolean }>('/api/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify({ token, new_password }),
    }),
  getProfile: () => request<Profile>('/api/profile'),
  updateProfile: (data: UpdateProfileRequest) =>
    request<Profile>('/api/profile', { method: 'PUT', body: JSON.stringify(data) }),
  monitors: (opts?: { tag?: string; customer?: string }) => {
    const params = new URLSearchParams()
    if (opts?.tag) params.set('tag', opts.tag)
    if (opts?.customer) params.set('customer', opts.customer)
    const q = params.toString()
    return request<Monitor[]>(q ? `/api/monitors?${q}` : '/api/monitors')
  },
  getMonitor: (id: string) => request<Monitor>(`/api/monitors/${id}`),
  createMonitor: (data: Partial<Monitor>) =>
    request<Monitor>('/api/monitors', { method: 'POST', body: JSON.stringify(data) }),
  updateMonitor: (id: string, data: Partial<Monitor>) =>
    request<Monitor>(`/api/monitors/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteMonitor: (id: string) => request(`/api/monitors/${id}`, { method: 'DELETE' }),
  results: (id: string, opts?: { limit?: number; offset?: number }) => {
    const limit = opts?.limit ?? 10
    const offset = opts?.offset ?? 0
    return request<PaginatedResults<CheckResult>>(
      `/api/monitors/${id}/results?limit=${limit}&offset=${offset}`,
    )
  },
  monitorIncidents: (id: string, opts?: { limit?: number; offset?: number }) => {
    const limit = opts?.limit ?? 20
    const offset = opts?.offset ?? 0
    return request<PaginatedResults<Incident>>(
      `/api/monitors/${id}/incidents?limit=${limit}&offset=${offset}`,
    )
  },
  stats: (id: string, period = '24h') =>
    request<MonitorStats>(`/api/monitors/${id}/stats?period=${period}`),
  performance: (period = '24h', customer?: string) => {
    const params = new URLSearchParams({ period })
    if (customer) params.set('customer', customer)
    return request<FleetPerformance>(`/api/performance?${params}`)
  },
  performanceTargets: (customer?: string) =>
    request<PerformanceTarget[]>(
      customer ? `/api/performance/targets?customer=${encodeURIComponent(customer)}` : '/api/performance/targets',
    ),
  getPerformanceTarget: (id: string) => request<PerformanceTarget>(`/api/performance/targets/${id}`),
  createPerformanceTarget: (data: Partial<PerformanceTarget>) =>
    request<PerformanceTarget>('/api/performance/targets', { method: 'POST', body: JSON.stringify(data) }),
  updatePerformanceTarget: (id: string, data: Partial<PerformanceTarget>) =>
    request<PerformanceTarget>(`/api/performance/targets/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deletePerformanceTarget: (id: string) => request(`/api/performance/targets/${id}`, { method: 'DELETE' }),
  performanceResults: (id: string) =>
    request<PerformanceResult[]>(`/api/performance/targets/${id}/results?limit=20`),
  performanceStats: (id: string, period = '24h') =>
    request<PerformanceStats>(`/api/performance/targets/${id}/stats?period=${period}`),
  listCustomers: () => request<Customer[]>('/api/settings/customers'),
  createCustomer: (data: { name: string; monitor_quota?: number }) =>
    request<Customer>('/api/settings/customers', { method: 'POST', body: JSON.stringify(data) }),
  updateCustomer: (id: string, data: { name: string; monitor_quota: number }) =>
    request<Customer>(`/api/settings/customers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteCustomer: (id: string) => request(`/api/settings/customers/${id}`, { method: 'DELETE' }),
  listTeam: () => request<TeamMember[]>('/api/settings/team'),
  createTeamMember: (data: CreateTeamMemberRequest) =>
    request<TeamMember>('/api/settings/team', { method: 'POST', body: JSON.stringify(data) }),
  updateTeamMember: (id: string, data: UpdateTeamMemberRequest) =>
    request<TeamMember>(`/api/settings/team/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteTeamMember: (id: string) => request(`/api/settings/team/${id}`, { method: 'DELETE' }),
  getGeneral: () => request<OrgSettings>('/api/settings/general'),
  putGeneral: (cfg: OrgSettings) =>
    request<OrgSettings>('/api/settings/general', { method: 'PUT', body: JSON.stringify(cfg) }),
  resetGeneral: () => request<OrgSettings>('/api/settings/general/reset', { method: 'POST' }),
  getSMTP: () => request<SMTPConfig>('/api/settings/smtp'),
  putSMTP: (cfg: SMTPConfig) =>
    request<SMTPConfig>('/api/settings/smtp', { method: 'PUT', body: JSON.stringify(cfg) }),
  testSMTP: (to: string) =>
    request('/api/settings/smtp/test', { method: 'POST', body: JSON.stringify({ to }) }),
  incidents: (openOnly = false) =>
    request<Incident[]>(`/api/incidents?limit=100${openOnly ? '&open=1' : ''}`),
  getWebhooks: () => request<WebhookConfig[]>('/api/settings/webhooks'),
  putWebhooks: (hooks: WebhookConfig[]) =>
    request<WebhookConfig[]>('/api/settings/webhooks', { method: 'PUT', body: JSON.stringify(hooks) }),
  listMaintenance: () => request<MaintenanceWindow[]>('/api/settings/maintenance'),
  createMaintenance: (data: Partial<MaintenanceWindow>) =>
    request<MaintenanceWindow>('/api/settings/maintenance', { method: 'POST', body: JSON.stringify(data) }),
  deleteMaintenance: (id: string) => request(`/api/settings/maintenance/${id}`, { method: 'DELETE' }),
  getServerSettings: () => request<ServerSettings>('/api/settings/server'),
  putServerSettings: (cfg: ServerSettings) =>
    request<ServerSettings>('/api/settings/server', { method: 'PUT', body: JSON.stringify(cfg) }),
  getStatusPageConfig: () => request<StatusPageConfig>('/api/settings/status-page'),
  putStatusPageConfig: (cfg: StatusPageConfig) =>
    request<StatusPageConfig>('/api/settings/status-page', { method: 'PUT', body: JSON.stringify(cfg) }),
  listAudit: () => request<AuditEntry[]>('/api/settings/audit'),
  listTokens: () => request<APIToken[]>('/api/settings/tokens'),
  createToken: (name: string) =>
    request<APITokenCreated>('/api/settings/tokens', { method: 'POST', body: JSON.stringify({ name }) }),
  deleteToken: (id: string) => request(`/api/settings/tokens/${id}`, { method: 'DELETE' }),
  publicStatus: () => request<PublicStatusResponse>('/api/public/status'),
}
