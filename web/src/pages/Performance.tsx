import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Area, AreaChart, Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import { api, FleetPerformance, PerformanceHealth, PerformanceTarget, ServicePerformance } from '../api'
import CustomerFilter, { matchesCustomerFilter } from '../components/CustomerFilter'
import { useAuth } from '../context/AuthContext'
import MetricCard from '../components/MetricCard'
import { colors } from '../theme'

const healthColors: Record<PerformanceHealth, string> = {
  good: colors.green,
  warning: colors.yellow,
  critical: colors.red,
  collecting: colors.textMuted,
}

const healthLabels: Record<PerformanceHealth, string> = {
  good: 'Within SLA',
  warning: 'Slow',
  critical: 'Slow',
  collecting: 'Collecting',
}

export default function Performance() {
  const { isAdmin, isPlatformAdmin } = useAuth()
  const [targets, setTargets] = useState<PerformanceTarget[]>([])
  const [summary, setSummary] = useState<FleetPerformance | null>(null)
  const [period, setPeriod] = useState('24h')
  const [search, setSearch] = useState('')
  const [selectedCustomers, setSelectedCustomers] = useState<string[]>([])
  const [customers, setCustomers] = useState<{ id: string; name: string }[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.listCustomers().then(c => setCustomers(c.map(x => ({ id: x.id, name: x.name })))).catch(() => {})
  }, [isPlatformAdmin])

  useEffect(() => {
    Promise.all([
      api.performanceTargets().then(setTargets),
      api.performance(period).then(setSummary),
    ]).catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [period])

  const scopedTargets = targets.filter(t => matchesCustomerFilter(t.tenant_id, selectedCustomers))
  const q = search.trim().toLowerCase()
  const filteredTargets = q
    ? scopedTargets.filter(t => `${t.name} ${t.url}`.toLowerCase().includes(q))
    : scopedTargets

  const targetIds = new Set(scopedTargets.map(t => t.id))
  const services = (summary?.services || []).filter(s => {
    if (!targetIds.has(s.service_id) && selectedCustomers.length > 0) return false
    if (!q) return true
    return `${s.name} ${s.target}`.toLowerCase().includes(q)
  })
  const attention = services.filter(s => s.health === 'warning')
  const ranked = services.filter(s => s.has_data)

  const timeline = (summary?.timeline || []).map(p => ({
    time: formatBucket(p.timestamp, period),
    avg: p.avg_ms,
  }))

  const rankingData = ranked.slice(0, 8).map(s => ({
    name: s.name.length > 18 ? `${s.name.slice(0, 16)}…` : s.name,
    p95: s.p95_ms,
    health: s.health,
  }))

  return (
    <div className="page">
      <div style={styles.topBar}>
        <div>
          <h1 className="page-title">Performance</h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            {filteredTargets.length === targets.length
              ? 'Measure website response time and latency — separate from uptime monitoring'
              : `${filteredTargets.length} of ${targets.length} targets`}
          </p>
        </div>
        <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
          {isPlatformAdmin && customers.length > 0 && (
            <CustomerFilter
              customers={customers}
              selectedIds={selectedCustomers}
              onChange={setSelectedCustomers}
            />
          )}
          <select value={period} onChange={e => setPeriod(e.target.value)} className="input" style={{ width: 'auto', padding: '8px 14px' }}>
            <option value="24h">Last 24 hours</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>
          {isAdmin && <Link to="/performance/targets/new" className="btn btn-primary">+ Add Target</Link>}
        </div>
      </div>

      {error && <div style={styles.error}>{error}</div>}

      {targets.length === 0 ? (
        <div style={styles.empty}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>▤</div>
          <div style={{ fontWeight: 600, marginBottom: 8 }}>No performance targets yet</div>
          <div style={{ color: colors.textMuted, marginBottom: 20, maxWidth: 440, margin: '0 auto 20px' }}>
            Add websites here to track response time, TTFB, and latency percentiles.
            These are independent from uptime monitors.
          </div>
          {isAdmin && <Link to="/performance/targets/new" className="btn btn-primary">Add Performance Target</Link>}
        </div>
      ) : (
        <>
          <div style={styles.searchRow}>
            <input
              className="input"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search by name or URL…"
              style={styles.searchInput}
            />
          </div>

          {filteredTargets.length === 0 ? (
            <div style={styles.empty}>
              <div style={{ fontWeight: 600, marginBottom: 8 }}>No matches</div>
              <div style={{ color: colors.textMuted }}>
                {search.trim()
                  ? `No targets match “${search.trim()}”.`
                  : 'No targets for the selected customers.'}
              </div>
            </div>
          ) : (
            <div style={styles.targetGrid}>
              {filteredTargets.map(t => (
                <Link key={t.id} to={`/performance/${t.id}`} style={styles.targetCard}>
                  <div style={styles.targetHeader}>
                    <span style={{ fontWeight: 600, fontSize: 15 }}>{t.name}</span>
                    <span style={{
                      fontSize: 11, fontWeight: 600,
                      color: t.last_status === 'up' ? colors.green : colors.yellow,
                    }}>
                      {t.latest_response_time_ms != null ? `${t.latest_response_time_ms} ms` : '—'}
                    </span>
                  </div>
                  <div style={{ color: colors.textMuted, fontSize: 12, marginTop: 6 }}>{t.url}</div>
                  <div style={{
                    display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap',
                    color: colors.textDim, fontSize: 11, marginTop: 8,
                  }}>
                    <span>Every {t.interval_seconds}s · SLA {t.slow_threshold_ms}ms</span>
                    {t.last_status !== 'up' && t.last_status !== 'unknown' && (
                      <span style={styles.slowBadge}>Slow</span>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          )}

          {summary && summary.total_checks > 0 && !q && selectedCustomers.length === 0 && (
            <>
              <div className="grid-4" style={{ marginBottom: 28 }}>
                <MetricCard label="Fleet P95" value={`${summary.p95_ms} ms`} accent="blue" />
                <MetricCard label="Fleet Avg" value={`${summary.avg_ms} ms`} />
                <MetricCard
                  label="Slow Rate"
                  value={`${((summary.slow_checks / summary.total_checks) * 100).toFixed(1)}%`}
                  accent={summary.slow_checks > 0 ? 'yellow' : 'green'}
                />
                <MetricCard label="Targets" value={String(targets.length)} accent="blue" />
              </div>

              <div style={styles.chartCard}>
                <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Fleet Latency Trend</h3>
                <div style={{ height: 260 }}>
                  {timeline.length > 0 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={timeline}>
                        <defs>
                          <linearGradient id="fleetGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={colors.brand} stopOpacity={0.35} />
                            <stop offset="100%" stopColor={colors.brand} stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid stroke={colors.border} strokeDasharray="3 3" vertical={false} />
                        <XAxis dataKey="time" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                        <YAxis unit="ms" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                        <Tooltip contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }} />
                        <Area type="monotone" dataKey="avg" stroke={colors.brand} fill="url(#fleetGrad)" strokeWidth={2} />
                      </AreaChart>
                    </ResponsiveContainer>
                  ) : (
                    <div style={styles.emptyChart}>Waiting for data…</div>
                  )}
                </div>
              </div>

              <div style={styles.twoCol}>
                {rankingData.length > 0 && (
                  <div style={styles.chartCard}>
                    <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>P95 Ranking</h3>
                    <div style={{ height: Math.max(180, rankingData.length * 40) }}>
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={rankingData} layout="vertical" margin={{ left: 8, right: 24 }}>
                          <CartesianGrid stroke={colors.border} strokeDasharray="3 3" horizontal={false} />
                          <XAxis type="number" unit="ms" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                          <YAxis type="category" dataKey="name" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} width={110} />
                          <Tooltip contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }} />
                          <Bar dataKey="p95" radius={[0, 4, 4, 0]} maxBarSize={16}>
                            {rankingData.map((entry, i) => (
                              <Cell key={i} fill={healthColors[entry.health]} />
                            ))}
                          </Bar>
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  </div>
                )}

                <div style={styles.chartCard}>
                  <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Latency Alerts</h3>
                  {attention.length === 0 ? (
                    <div style={{ color: colors.green, fontSize: 14, padding: '16px 0' }}>All targets within SLA</div>
                  ) : (
                    attention.map(s => <AlertRow key={s.service_id} service={s} />)
                  )}
                </div>
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}

function AlertRow({ service: s }: { service: ServicePerformance }) {
  const slowPct = s.check_count > 0 ? ((s.slow_count / s.check_count) * 100).toFixed(1) : '0.0'
  const detail = s.health === 'warning'
    ? `${slowPct}% of checks slow (P95 ${s.p95_ms} ms · SLA ${s.slow_threshold_ms} ms)`
    : `P95 ${s.p95_ms} ms`
  return (
    <Link to={`/performance/${s.service_id}`} style={styles.alertRow}>
      <span style={{ ...styles.dot, background: healthColors[s.health] }} />
      <div>
        <div style={{ fontWeight: 600, fontSize: 14 }}>{s.name}</div>
        <div style={{ color: colors.textMuted, fontSize: 12 }}>{detail}</div>
      </div>
    </Link>
  )
}

function formatBucket(iso: string, period: string) {
  const d = new Date(iso)
  if (period === '24h') return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

const styles: Record<string, React.CSSProperties> = {
  topBar: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24, gap: 16, flexWrap: 'wrap' },
  searchRow: { marginBottom: 20 },
  searchInput: { maxWidth: 420, width: '100%' },
  slowBadge: {
    color: colors.yellow,
    background: colors.yellowDim,
    padding: '2px 8px',
    borderRadius: 999,
    fontWeight: 600,
    fontSize: 10,
    letterSpacing: '0.02em',
  },
  targetGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 14, marginBottom: 28 },
  targetCard: {
    display: 'block', padding: '18px 20px', borderRadius: 12,
    background: colors.card, border: `1px solid ${colors.border}`,
    color: colors.text, textDecoration: 'none',
  },
  targetHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  chartCard: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: '20px 24px', marginBottom: 20 },
  twoCol: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))', gap: 20 },
  emptyChart: { color: colors.textMuted, textAlign: 'center', paddingTop: 80 },
  empty: { textAlign: 'center', padding: '64px 24px', background: colors.card, borderRadius: 12, border: `1px solid ${colors.border}` },
  alertRow: { display: 'flex', gap: 12, alignItems: 'center', padding: '10px 0', color: colors.text, textDecoration: 'none', borderBottom: `1px solid ${colors.border}` },
  dot: { width: 8, height: 8, borderRadius: '50%', flexShrink: 0 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
