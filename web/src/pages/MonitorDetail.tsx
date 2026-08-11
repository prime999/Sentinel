import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Area, AreaChart, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts'
import { api, CheckResult, DNSDetails, Incident, Monitor, MonitorStats, NotificationsSummary, PortDetails, SSLDetails } from '../api'
import DeleteMonitorButton from '../components/DeleteMonitorButton'
import IncidentFilters, { IncidentFilterValues } from '../components/IncidentFilters'
import IncidentStatus, { incidentStatusLabel } from '../components/IncidentStatus'
import MetricCard from '../components/MetricCard'
import StatusBadge from '../components/StatusBadge'
import TypeBadge from '../components/TypeBadge'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'
import { secondsUntilNextCheck, useAdaptivePoll } from '../utils/poll'

export default function MonitorDetail() {
  const { isAdmin } = useAuth()
  const { id } = useParams<{ id: string }>()
  const [monitor, setMonitor] = useState<Monitor | null>(null)
  const [stats, setStats] = useState<MonitorStats | null>(null)
  const [latest, setLatest] = useState<CheckResult | undefined>()
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [period, setPeriod] = useState('24h')
  const periodRef = useRef(period)
  periodRef.current = period

  const load = useCallback(async () => {
    if (!id) return null
    const [m, r, s, inc] = await Promise.all([
      api.getMonitor(id),
      api.results(id, { limit: 1, offset: 0 }),
      api.stats(id, periodRef.current),
      api.monitorIncidents(id, { limit: 20, offset: 0 }),
    ])
    setMonitor(m)
    setLatest(r.items[0])
    setStats(s)
    setIncidents(inc.items)
    return m
  }, [id])

  const refreshRef = useAdaptivePoll(id, load, [period])

  if (!monitor) return <div style={{ color: colors.textMuted }}>Loading…</div>

  const type = monitor.type || 'http'
  const target = type === 'port' ? `${monitor.url}:${monitor.port}` : monitor.url
  const chartData = (stats?.points || []).map(p => ({
    time: new Date(p.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    ms: p.response_time_ms,
  }))

  return (
    <div className="page">
      {/* Header */}
      <div style={styles.header}>
        <div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginBottom: 6 }}>
            <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>{target}</h1>
            <TypeBadge type={type} url={monitor.url} />
            <StatusBadge status={monitor.last_status} />
            {monitor.invert && <span style={styles.invertBadge}>Inverted</span>}
          </div>
          <div style={{ color: colors.textMuted, fontSize: 14 }}>{monitor.name}</div>
        </div>
        {isAdmin && (
          <div style={{ display: 'flex', gap: 10 }}>
            <Link to={`/monitors/${id}/edit`} className="btn">Edit</Link>
            <DeleteMonitorButton id={monitor.id} name={monitor.name} variant="danger" />
          </div>
        )}
      </div>

      <TypeMetrics monitor={monitor} stats={stats} latest={latest} />

      {type !== 'dns' && (
        <div style={styles.chartCard}>
          <div style={styles.chartHeader}>
            <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>Availability History</h3>
            <select value={period} onChange={e => setPeriod(e.target.value)} className="input" style={{ width: 'auto', padding: '6px 12px' }}>
              <option value="24h">Last 24 hours</option>
              <option value="7d">Last 7 days</option>
              <option value="30d">Last 30 days</option>
            </select>
          </div>
          <div style={{ height: 260 }}>
            {chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="fillTeal" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={colors.brand} stopOpacity={0.4} />
                      <stop offset="100%" stopColor={colors.brand} stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid stroke={colors.border} strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="time" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                  <YAxis unit="ms" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                  <Tooltip
                    contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }}
                    labelStyle={{ color: colors.textMuted }}
                  />
                  <Area type="monotone" dataKey="ms" stroke={colors.brand} fill="url(#fillTeal)" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div style={styles.emptyChart}>No data yet — waiting for first check</div>
            )}
          </div>
        </div>
      )}

      {stats && type === 'http' && (
        <div style={{ ...styles.chartCard, marginBottom: 20 }}>
          <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Uptime</h3>
          <UptimeBar label="Selected period" pct={stats.uptime_pct} />
        </div>
      )}

      <div className="detail-grid" style={{ marginBottom: 20 }}>
        <TypeDetailPanel monitor={monitor} latest={latest} />
        <SidePanel monitor={monitor} incidents={incidents} onCheckDue={() => refreshRef.current?.()} />
      </div>

      <IncidentsTable monitorId={monitor.id} />
    </div>
  )
}

function TypeMetrics({ monitor, stats, latest }: { monitor: Monitor; stats: MonitorStats | null; latest?: CheckResult }) {
  const type = monitor.type || 'http'

  if (type === 'ssl') {
    const ssl = parseSSL(latest?.details)
    return (
      <div className="grid-4" style={{ marginBottom: 24 }}>
        <MetricCard label="Days Remaining" value={ssl ? String(ssl.days_remaining) : '—'} accent={ssl && ssl.days_remaining <= 30 ? 'yellow' : 'green'} />
        <MetricCard label="Expires" value={ssl ? formatDate(ssl.expires_at) : '—'} />
        <MetricCard label="Issuer" value={ssl?.issuer || '—'} accent="blue" />
        <MetricCard label="Certificate" value={ssl?.issues?.length ? `${ssl.issues.length} issue(s)` : 'Valid'} accent={ssl?.issues?.length ? 'red' : 'green'} />
      </div>
    )
  }

  if (type === 'port') {
    const port = parsePort(latest?.details)
    return (
      <div className="grid-4" style={{ marginBottom: 24 }}>
        <MetricCard label="Status" value={port?.open ? 'OPEN' : port ? 'CLOSED' : '—'} accent={port?.open ? 'green' : 'red'} />
        <MetricCard label="Last Check" value={monitor.last_checked_at ? timeAgo(monitor.last_checked_at) : '—'} />
        <MetricCard label="Interval" value={`${monitor.interval_seconds}s`} />
        <MetricCard label="Uptime" value={`${(stats?.uptime_pct ?? 0).toFixed(1)}%`} accent="green" />
      </div>
    )
  }

  if (type === 'dns') {
    const dns = parseDNS(latest?.details)
    const total = dns ? Object.values(dns.records).reduce((n, v) => n + v.length, 0) : 0
    return (
      <div className="grid-4" style={{ marginBottom: 24 }}>
        <MetricCard label="Record Types" value={dns ? String(Object.keys(dns.records).length) : '—'} accent="blue" />
        <MetricCard label="Total Records" value={total ? String(total) : '—'} />
        <MetricCard label="Changes" value={dns?.changes?.length ? String(dns.changes.length) : 'None'} accent={dns?.changes?.length ? 'yellow' : 'default'} />
        <MetricCard label="Interval" value={`${monitor.interval_seconds}s`} />
      </div>
    )
  }

  return (
    <div className="grid-4" style={{ marginBottom: 24 }}>
      <MetricCard label="Status" value={monitor.last_status.toUpperCase()} accent={monitor.last_status === 'up' ? 'green' : monitor.last_status === 'down' ? 'red' : 'yellow'} />
      <MetricCard label="Last Check" value={monitor.last_checked_at ? timeAgo(monitor.last_checked_at) : '—'} />
      <MetricCard label="Interval" value={`${monitor.interval_seconds}s`} />
      <MetricCard label="Uptime" value={`${(stats?.uptime_pct ?? 0).toFixed(1)}%`} accent="green" />
    </div>
  )
}

function UptimeBar({ label, pct }: { label: string; pct: number }) {
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 6 }}>
        <span style={{ color: colors.textMuted }}>{label}</span>
        <span style={{ fontWeight: 600 }}>{pct.toFixed(1)}%</span>
      </div>
      <div style={{ height: 8, background: colors.bg, borderRadius: 4, overflow: 'hidden' }}>
        <div style={{ height: '100%', width: `${Math.min(pct, 100)}%`, background: colors.green, borderRadius: 4, transition: 'width 0.3s' }} />
      </div>
    </div>
  )
}

function TypeDetailPanel({ monitor, latest }: { monitor: Monitor; latest?: CheckResult }) {
  const type = monitor.type || 'http'

  if (type === 'ssl') {
    const ssl = parseSSL(latest?.details)
    if (!ssl) return <Panel title="Certificate Details"><Empty /></Panel>
    return (
      <Panel title="Certificate Details">
        <Row label="Subject" value={ssl.subject} />
        <Row label="Issuer" value={ssl.issuer} />
        <Row label="Expires" value={`${formatDate(ssl.expires_at)} (${ssl.days_remaining} days)`} />
        <Row label="Fingerprint" value={ssl.fingerprint} mono />
        {ssl.issues && ssl.issues.length > 0 && (
          <div style={{ marginTop: 12, padding: 10, background: colors.redDim, borderRadius: 8, color: colors.red, fontSize: 13 }}>
            Issues: {ssl.issues.join(', ')}
          </div>
        )}
      </Panel>
    )
  }

  if (type === 'dns') {
    const dns = parseDNS(latest?.details)
    if (!dns) return <Panel title="DNS Records"><Empty /></Panel>
    return (
      <Panel title="DNS Records">
        {Object.entries(dns.records).map(([rt, vals]) => (
          <Row key={rt} label={rt} value={vals.length ? vals.join(', ') : '—'} />
        ))}
        {dns.changes && dns.changes.length > 0 && (
          <div style={{ marginTop: 12, padding: 10, background: colors.yellowDim, borderRadius: 8, fontSize: 13 }}>
            {dns.changes.map((c, i) => (
              <div key={i} style={{ marginTop: i ? 4 : 0 }}>{c.type}: {c.before || '∅'} → {c.after || '∅'}</div>
            ))}
          </div>
        )}
      </Panel>
    )
  }

  if (type === 'port') {
    const port = parsePort(latest?.details)
    return (
      <Panel title="Connection Details">
        <Row label="Host" value={port?.host || monitor.url} />
        <Row label="Port" value={String(port?.port ?? monitor.port ?? '—')} />
        <Row label="Protocol" value="TCP" />
        <Row label="Timeout" value={`${monitor.timeout_ms} ms`} />
        <Row label="Status" value={port?.open ? 'Open' : port ? 'Closed' : '—'} />
      </Panel>
    )
  }

  return (
    <Panel title="Connection Details">
      <Row label="URL" value={monitor.url} />
      <Row label="Method" value={monitor.method} />
      <Row label="Status Code" value={latest?.status_code != null ? String(latest.status_code) : '—'} />
      <Row label="Timeout" value={`${monitor.timeout_ms} ms`} />
      <Row label="Interval" value={`${monitor.interval_seconds}s`} />
      {latest && (
        <>
          <div style={{ borderTop: `1px solid ${colors.border}`, margin: '12px 0' }} />
          <Row label="DNS" value={latest.dns_ms != null ? `${latest.dns_ms} ms` : '—'} />
          <Row label="TCP" value={latest.tcp_ms != null ? `${latest.tcp_ms} ms` : '—'} />
          <Row label="TLS" value={latest.tls_ms != null ? `${latest.tls_ms} ms` : '—'} />
          <Row label="TTFB" value={latest.ttfb_ms != null ? `${latest.ttfb_ms} ms` : '—'} />
        </>
      )}
    </Panel>
  )
}

function NextCheckCountdown({ monitor, onDue }: { monitor: Monitor; onDue?: () => void }) {
  const monitorRef = useRef(monitor)
  monitorRef.current = monitor
  const onDueRef = useRef(onDue)
  onDueRef.current = onDue
  const [seconds, setSeconds] = useState(() => secondsUntilNextCheck(monitor))

  useEffect(() => {
    setSeconds(secondsUntilNextCheck(monitor))
    const tick = setInterval(() => {
      const next = secondsUntilNextCheck(monitorRef.current)
      setSeconds(prev => {
        if (prev > 0 && next === 0) onDueRef.current?.()
        return next
      })
    }, 1000)
    return () => clearInterval(tick)
  }, [monitor.last_checked_at, monitor.interval_seconds, monitor.enabled])

  if (!monitor.enabled) {
    return (
      <>
        <div style={{ fontSize: 36, fontWeight: 700, color: colors.textMuted }}>—</div>
        <div style={{ fontSize: 12, color: colors.textMuted, marginTop: 4 }}>monitor disabled</div>
      </>
    )
  }

  return (
    <>
      <div style={{ fontSize: 36, fontWeight: 700, color: colors.brand }}>
        {seconds === 0 ? 'Checking…' : `${seconds}s`}
      </div>
      <div style={{ fontSize: 12, color: colors.textMuted, marginTop: 4 }}>
        {seconds === 0 ? 'probe due now' : 'until next probe'}
      </div>
    </>
  )
}

function SidePanel({ monitor, incidents, onCheckDue }: {
  monitor: Monitor
  incidents: Incident[]
  onCheckDue?: () => void
}) {
  const { isPlatformAdmin } = useAuth()
  const [summary, setSummary] = useState<NotificationsSummary | null>(null)
  const lastIncident = incidents.find(i => i.type !== 'recovery') || incidents[0]

  useEffect(() => {
    api.getNotificationsSummary().then(setSummary).catch(() => setSummary(null))
  }, [])

  return (
    <div style={{ display: 'grid', gap: 16 }}>
      <Panel title="Last Incident">
        {lastIncident ? (
          <>
            <Row label="Type" value={lastIncident.type} />
            <Row label="When" value={new Date(lastIncident.started_at).toLocaleString()} />
            <Row label="Message" value={lastIncident.message || '—'} />
            <Row label="Status" value={incidentStatusLabel(lastIncident)} />
          </>
        ) : (
          <div style={{ color: colors.textMuted, fontSize: 13 }}>No recent incidents</div>
        )}
      </Panel>
      <Panel title="Notifications">
        <Row
          label="Email"
          value={channelStatusLabel({
            globalConfigured: !!summary?.email?.configured,
            globalEnabled: !!summary?.email?.enabled,
            monitorEnabled: monitor.notify_email !== false,
            detail: monitor.alert_emails ? 'custom recipients' : 'default recipients',
          })}
        />
        <Row
          label="Slack"
          value={channelStatusLabel({
            globalConfigured: !!summary?.slack?.configured,
            globalEnabled: !!summary?.slack?.enabled,
            monitorEnabled: monitor.notify_slack !== false,
          })}
        />
        {isPlatformAdmin && (
          <Row
            label="Webhooks"
            value={channelStatusLabel({
              globalConfigured: !!summary?.webhooks?.configured,
              globalEnabled: !!summary?.webhooks?.enabled,
              monitorEnabled: monitor.notify_webhooks !== false,
            })}
          />
        )}
      </Panel>
      <Panel title="Next Check">
        <div style={{ textAlign: 'center', padding: '12px 0' }}>
          <NextCheckCountdown monitor={monitor} onDue={onCheckDue} />
        </div>
      </Panel>
    </div>
  )
}

function channelStatusLabel(opts: {
  globalConfigured: boolean
  globalEnabled: boolean
  monitorEnabled: boolean
  detail?: string
}): string {
  if (!opts.globalConfigured) return 'Not configured'
  if (!opts.globalEnabled) return 'Off (global)'
  if (!opts.monitorEnabled) return 'Off (this monitor)'
  return opts.detail ? `On (${opts.detail})` : 'On'
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={styles.panel}>
      <h3 style={{ margin: '0 0 16px', fontSize: 14, fontWeight: 600, color: colors.textMuted, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{title}</h3>
      {children}
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '8px 0', borderBottom: `1px solid ${colors.border}`, fontSize: 13 }}>
      <span style={{ color: colors.textMuted, flexShrink: 0 }}>{label}</span>
      <span style={{ fontWeight: 500, textAlign: 'right', fontFamily: mono ? 'monospace' : 'inherit', fontSize: mono ? 11 : 13, wordBreak: 'break-all' }}>{value}</span>
    </div>
  )
}

function Empty() {
  return <div style={{ color: colors.textMuted, fontSize: 13 }}>Waiting for check data…</div>
}

function IncidentsTable({ monitorId }: { monitorId: string }) {
  const pageSize = 10
  const [page, setPage] = useState(0)
  const [filters, setFilters] = useState<IncidentFilterValues>({
    date: '',
    status: '',
    type: '',
    monitorId: '',
  })
  const [items, setItems] = useState<Incident[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setPage(0)
  }, [monitorId, filters.date, filters.status, filters.type])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.monitorIncidents(monitorId, {
      limit: pageSize,
      offset: page * pageSize,
      date: filters.date || undefined,
      status: filters.status || undefined,
      type: filters.type || undefined,
    })
      .then(res => {
        if (cancelled) return
        setItems(res.items)
        setTotal(res.total)
      })
      .catch(() => {
        if (!cancelled) {
          setItems([])
          setTotal(0)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [monitorId, page, filters.date, filters.status, filters.type])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const from = total === 0 ? 0 : page * pageSize + 1
  const to = Math.min(total, (page + 1) * pageSize)
  const filtered = !!(filters.date || filters.status || filters.type)

  return (
    <div style={{ ...styles.chartCard, marginTop: 0 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16, gap: 12, flexWrap: 'wrap' }}>
        <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>Recent Incidents</h3>
        <span style={{ fontSize: 13, color: colors.textMuted }}>
          {total === 0
            ? (filtered ? 'No matching incidents' : 'No incidents yet')
            : `Showing ${from}–${to} of ${total}`}
        </span>
      </div>
      <div style={{ marginBottom: 16 }}>
        <IncidentFilters
          value={filters}
          onChange={setFilters}
          showMonitor={false}
        />
      </div>
      <div style={{ overflowX: 'auto' }}>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>Started</th>
              <th style={styles.th}>Type</th>
              <th style={styles.th}>Message</th>
              <th style={styles.th}>Status</th>
              <th style={styles.th}>Resolved</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} style={{ ...styles.td, color: colors.textMuted }}>Loading…</td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ ...styles.td, color: colors.textMuted }}>No incidents recorded for this monitor.</td>
              </tr>
            ) : (
              items.map(inc => (
                <tr key={inc.id}>
                  <td style={styles.td}>{new Date(inc.started_at).toLocaleString()}</td>
                  <td style={styles.td}>
                    <span style={styles.incidentType}>{inc.type}</span>
                  </td>
                  <td style={{ ...styles.td, color: colors.textMuted, maxWidth: 360 }}>
                    {inc.message || '—'}
                  </td>
                  <td style={styles.td}>
                    <IncidentStatus incident={inc} />
                  </td>
                  <td style={{ ...styles.td, color: colors.textMuted }}>
                    {inc.resolved_at ? new Date(inc.resolved_at).toLocaleString() : '—'}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {total > pageSize && (
        <div style={styles.pager}>
          <button
            type="button"
            className="btn"
            style={styles.pagerBtn}
            disabled={page <= 0 || loading}
            onClick={() => setPage(p => Math.max(0, p - 1))}
          >
            Previous
          </button>
          <span style={{ fontSize: 13, color: colors.textMuted }}>
            Page {page + 1} of {totalPages}
          </span>
          <button
            type="button"
            className="btn"
            style={styles.pagerBtn}
            disabled={page + 1 >= totalPages || loading}
            onClick={() => setPage(p => p + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

function parseSSL(d?: string): SSLDetails | null { try { return d ? JSON.parse(d) : null } catch { return null } }
function parseDNS(d?: string): DNSDetails | null { try { return d ? JSON.parse(d) : null } catch { return null } }
function parsePort(d?: string): PortDetails | null { try { return d ? JSON.parse(d) : null } catch { return null } }
function formatDate(iso: string) { try { return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }) } catch { return iso } }
function timeAgo(iso: string) { const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000); return s < 60 ? `${s}s ago` : s < 3600 ? `${Math.floor(s / 60)}m ago` : `${Math.floor(s / 3600)}h ago` }

const styles: Record<string, React.CSSProperties> = {
  header: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
    marginBottom: 24, paddingBottom: 20, borderBottom: `1px solid ${colors.border}`,
  },
  invertBadge: {
    fontSize: 11, fontWeight: 600, color: colors.yellow,
    background: 'rgba(210,153,34,0.15)', padding: '2px 8px', borderRadius: 4,
  },
  chartCard: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '20px 24px', marginBottom: 20,
  },
  chartHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 },
  emptyChart: { color: colors.textMuted, textAlign: 'center', paddingTop: 120, fontSize: 14 },
  panel: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '18px 20px',
  },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 13 },
  th: {
    textAlign: 'left', padding: '10px 12px', borderBottom: `1px solid ${colors.border}`,
    color: colors.textMuted, fontWeight: 600, fontSize: 12, textTransform: 'uppercase',
  },
  td: { padding: '12px', borderBottom: `1px solid ${colors.border}` },
  incidentType: {
    textTransform: 'uppercase',
    fontSize: 11,
    fontWeight: 700,
    color: colors.textMuted,
    letterSpacing: '0.04em',
  },
  pager: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: 12,
    marginTop: 16,
    paddingTop: 12,
    borderTop: `1px solid ${colors.border}`,
  },
  pagerBtn: {
    minHeight: 36,
    padding: '0 14px',
    fontSize: 13,
  },
}
