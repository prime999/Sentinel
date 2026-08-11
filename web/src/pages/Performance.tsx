import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import { api, FleetPerformance, PerformanceHealth, PerformanceTarget, ServicePerformance } from '../api'
import CustomerFilter, { matchesCustomerFilter } from '../components/CustomerFilter'
import ConfirmDialog from '../components/ConfirmDialog'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'

type HealthTab = 'all' | 'good' | 'slow' | 'collecting'

const healthColors: Record<PerformanceHealth, string> = {
  good: colors.green,
  warning: colors.yellow,
  critical: colors.red,
  collecting: colors.textMuted,
}

function timeAgo(iso: string): string {
  const sec = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`
  return new Date(iso).toLocaleDateString()
}

function formatBucket(iso: string, period: string) {
  const d = new Date(iso)
  if (period === '24h') return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function targetHealth(t: PerformanceTarget, svc?: ServicePerformance): PerformanceHealth {
  if (svc?.has_data) return svc.health
  const sla = t.slow_threshold_ms || 0
  const latest = t.latest_response_time_ms
  if (sla > 0 && latest != null && latest >= sla) return 'warning'
  if (t.last_checked_at) return 'good'
  return 'collecting'
}

function healthLabel(h: PerformanceHealth): string {
  if (h === 'good') return 'Within SLA'
  if (h === 'warning' || h === 'critical') return 'Slow'
  return 'Collecting'
}

export default function Performance() {
  const { isAdmin, isPlatformAdmin } = useAuth()
  const [targets, setTargets] = useState<PerformanceTarget[]>([])
  const [summary, setSummary] = useState<FleetPerformance | null>(null)
  const [period, setPeriod] = useState('24h')
  const [search, setSearch] = useState('')
  const [healthTab, setHealthTab] = useState<HealthTab>('all')
  const [selectedCustomers, setSelectedCustomers] = useState<string[]>([])
  const [customers, setCustomers] = useState<{ id: string; name: string }[]>([])
  const [error, setError] = useState('')
  const [menuId, setMenuId] = useState<string | null>(null)
  const [deleteTargetRow, setDeleteTargetRow] = useState<PerformanceTarget | null>(null)
  const [deleting, setDeleting] = useState(false)
  const menuRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.listCustomers().then(c => setCustomers(c.map(x => ({ id: x.id, name: x.name })))).catch(() => {})
  }, [isPlatformAdmin])

  useEffect(() => {
    Promise.all([
      api.performanceTargets().then(setTargets),
      api.performance(period).then(setSummary),
    ]).catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
    const id = setInterval(() => {
      api.performanceTargets().then(setTargets).catch(() => {})
      api.performance(period).then(setSummary).catch(() => {})
    }, 30000)
    return () => clearInterval(id)
  }, [period])

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!menuRef.current?.contains(e.target as Node)) setMenuId(null)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [])

  const serviceById = useMemo(() => {
    const map: Record<string, ServicePerformance> = {}
    for (const s of summary?.services || []) map[s.service_id] = s
    return map
  }, [summary])

  const scopedTargets = useMemo(
    () => targets.filter(t => matchesCustomerFilter(t.tenant_id, selectedCustomers)),
    [targets, selectedCustomers],
  )

  const q = search.trim().toLowerCase()
  const searched = useMemo(() => {
    if (!q) return scopedTargets
    return scopedTargets.filter(t => `${t.name} ${t.url}`.toLowerCase().includes(q))
  }, [scopedTargets, q])

  const withHealth = useMemo(
    () => searched.map(t => ({ target: t, health: targetHealth(t, serviceById[t.id]) })),
    [searched, serviceById],
  )

  const filtered = useMemo(() => {
    if (healthTab === 'all') return withHealth
    if (healthTab === 'slow') return withHealth.filter(r => r.health === 'warning' || r.health === 'critical')
    return withHealth.filter(r => r.health === healthTab)
  }, [withHealth, healthTab])

  const good = withHealth.filter(r => r.health === 'good').length
  const slow = withHealth.filter(r => r.health === 'warning' || r.health === 'critical').length
  const collecting = withHealth.filter(r => r.health === 'collecting').length

  const attention = (summary?.services || []).filter(s => {
    if (selectedCustomers.length > 0 && !scopedTargets.some(t => t.id === s.service_id)) return false
    return s.health === 'warning' || s.health === 'critical'
  })

  const timeline = (summary?.timeline || []).map(p => ({
    time: formatBucket(p.timestamp, period),
    avg: p.avg_ms,
  }))

  const slowRate = summary && summary.total_checks > 0
    ? ((summary.slow_checks / summary.total_checks) * 100).toFixed(1)
    : null

  const healthTabs: { id: HealthTab; label: string; count: number }[] = [
    { id: 'all', label: 'All', count: withHealth.length },
    { id: 'good', label: 'Within SLA', count: good },
    { id: 'slow', label: 'Slow', count: slow },
    { id: 'collecting', label: 'Collecting', count: collecting },
  ]

  async function confirmDelete() {
    if (!deleteTargetRow) return
    setDeleting(true)
    try {
      await api.deletePerformanceTarget(deleteTargetRow.id)
      setTargets(prev => prev.filter(t => t.id !== deleteTargetRow.id))
      setDeleteTargetRow(null)
      setMenuId(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="page" style={styles.page}>
      <ConfirmDialog
        open={!!deleteTargetRow}
        title="Delete performance target"
        message={deleteTargetRow ? `Delete “${deleteTargetRow.name}”? Latency history for this target will be removed.` : ''}
        confirmLabel="Delete"
        danger
        busy={deleting}
        onConfirm={confirmDelete}
        onCancel={() => { if (!deleting) setDeleteTargetRow(null) }}
      />
      <div style={styles.layout}>
        <div style={styles.main}>
          <div style={styles.topBar}>
            <div>
              <h1 className="page-title" style={{ fontSize: 28 }}>Performance</h1>
              <p className="page-subtitle" style={{ marginBottom: 0 }}>
                Measure website response time and latency — separate from uptime monitoring.
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
              <select
                value={period}
                onChange={e => setPeriod(e.target.value)}
                className="input"
                style={{ width: 'auto', padding: '8px 14px' }}
              >
                <option value="24h">Last 24 hours</option>
                <option value="7d">Last 7 days</option>
                <option value="30d">Last 30 days</option>
              </select>
              {isAdmin && (
                <Link to="/performance/targets/new" className="btn btn-primary">+ Add Target</Link>
              )}
            </div>
          </div>

          {searched.length > 0 && (
            <div style={styles.metrics}>
              <div style={styles.metricWide}>
                <div style={styles.metricLabel}>Fleet P95</div>
                <div style={styles.metricValue}>
                  {summary?.p95_ms != null ? `${summary.p95_ms} ms` : '—'}
                </div>
                <div style={styles.metricSub}>
                  {period === '24h' ? 'Last 24 hours' : period === '7d' ? 'Last 7 days' : 'Last 30 days'}
                  {slowRate != null ? ` · ${slowRate}% slow` : ''}
                </div>
              </div>
              <div style={styles.metricCard}>
                <div style={styles.metricLabel}>Targets</div>
                <div style={styles.metricValue}>{searched.length}</div>
                <div style={styles.metricSub}>Total targets</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(34,197,94,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.green }}>{good}</div>
                <div style={styles.statLabel}>Within SLA</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(245,158,11,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.yellow }}>{slow}</div>
                <div style={styles.statLabel}>Slow</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(156,163,175,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.textMuted }}>{collecting}</div>
                <div style={styles.statLabel}>Collecting</div>
              </div>
            </div>
          )}

          {(targets.length > 0 || search) && (
            <div style={styles.toolbar}>
              <input
                className="input"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search targets…"
                style={styles.searchInput}
              />
              <div style={styles.tabs}>
                {healthTabs.map(tab => (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => setHealthTab(tab.id)}
                    style={{
                      ...styles.tab,
                      ...(healthTab === tab.id ? styles.tabActive : {}),
                    }}
                  >
                    {tab.label}
                    <span style={styles.tabCount}>{tab.count}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {error && <div style={styles.error}>{error}</div>}

          {targets.length === 0 ? (
            <div style={styles.empty}>
              <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 8 }}>No performance targets yet</div>
              <div style={{ color: colors.textMuted, marginBottom: 20 }}>
                Add websites here to track response time and latency percentiles.
              </div>
              {isAdmin && <Link to="/performance/targets/new" className="btn btn-primary">Add Target</Link>}
            </div>
          ) : filtered.length === 0 ? (
            <div style={styles.empty}>
              <div style={{ fontWeight: 600, marginBottom: 8 }}>No matches</div>
              <div style={{ color: colors.textMuted }}>
                {search.trim()
                  ? `No targets match “${search.trim()}”.`
                  : 'No targets for the selected filters.'}
              </div>
            </div>
          ) : (
            <div style={styles.tableWrap}>
              <table style={styles.table}>
                <colgroup>
                  <col style={{ width: '28%' }} />
                  <col style={{ width: '14%' }} />
                  <col style={{ width: '14%' }} />
                  <col style={{ width: '12%' }} />
                  <col style={{ width: '12%' }} />
                  <col style={{ width: '14%' }} />
                  <col style={{ width: '48px' }} />
                </colgroup>
                <thead>
                  <tr>
                    <th style={styles.th}>Target</th>
                    <th style={styles.th}>Status</th>
                    <th style={styles.th}>Latency</th>
                    <th style={styles.th}>P95</th>
                    <th style={styles.th}>SLA</th>
                    <th style={styles.th}>Last Checked</th>
                    <th style={{ ...styles.th, width: 48 }} />
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(({ target: t, health }) => {
                    const svc = serviceById[t.id]
                    const ms = t.latest_response_time_ms
                    return (
                      <tr key={t.id} style={styles.tr}>
                        <td style={styles.td}>
                          <Link to={`/performance/${t.id}`} style={styles.targetLink}>
                            <span style={styles.targetName}>{t.name}</span>
                            <span style={styles.targetUrl}>{t.url}</span>
                          </Link>
                        </td>
                        <td style={styles.td}>
                          <span style={{
                            ...styles.healthBadge,
                            color: healthColors[health],
                            background: `${healthColors[health]}22`,
                          }}>
                            <span style={{ ...styles.healthDot, background: healthColors[health] }} />
                            {healthLabel(health)}
                          </span>
                        </td>
                        <td style={styles.td}>
                          <span style={{
                            fontWeight: 600,
                            color: health === 'good' || health === 'collecting' ? colors.text : colors.yellow,
                          }}>
                            {typeof ms === 'number' ? `${ms} ms` : '—'}
                          </span>
                        </td>
                        <td style={styles.td}>
                          {svc?.has_data ? `${svc.p95_ms} ms` : '—'}
                        </td>
                        <td style={{ ...styles.td, color: colors.textMuted }}>
                          {t.slow_threshold_ms} ms
                        </td>
                        <td style={{ ...styles.td, color: colors.textMuted }}>
                          {t.last_checked_at ? timeAgo(t.last_checked_at) : 'Waiting'}
                        </td>
                        <td style={styles.td}>
                          <div style={{ position: 'relative' }} ref={menuId === t.id ? menuRef : undefined}>
                            <button
                              type="button"
                              aria-label="Actions"
                              style={styles.kebab}
                              onClick={() => setMenuId(menuId === t.id ? null : t.id)}
                            >
                              ⋮
                            </button>
                            {menuId === t.id && (
                              <div style={styles.menu}>
                                <Link
                                  to={`/performance/${t.id}`}
                                  style={styles.menuItem}
                                  onClick={() => setMenuId(null)}
                                >
                                  View
                                </Link>
                                {isAdmin && (
                                  <>
                                    <Link
                                      to={`/performance/targets/${t.id}/edit`}
                                      style={styles.menuItem}
                                      onClick={() => setMenuId(null)}
                                    >
                                      Edit
                                    </Link>
                                    <button
                                      type="button"
                                      style={styles.menuDangerBtn}
                                      onClick={() => {
                                        setMenuId(null)
                                        setDeleteTargetRow(t)
                                      }}
                                    >
                                      Delete
                                    </button>
                                  </>
                                )}
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <aside style={styles.rail}>
          <section style={styles.railCard}>
            <div style={styles.railLabel}>Fleet Latency</div>
            <div style={{ height: 140, marginTop: 8 }}>
              {timeline.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={timeline}>
                    <defs>
                      <linearGradient id="perfRailGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={colors.brand} stopOpacity={0.35} />
                        <stop offset="100%" stopColor={colors.brand} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke={colors.border} strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="time" hide />
                    <YAxis hide domain={['auto', 'auto']} />
                    <Tooltip
                      contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }}
                      formatter={(v: number) => [`${v} ms`, 'Avg']}
                    />
                    <Area type="monotone" dataKey="avg" stroke={colors.brand} fill="url(#perfRailGrad)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div style={styles.railEmpty}>Waiting for data…</div>
              )}
            </div>
            <div style={styles.railMeta}>
              Avg {summary?.avg_ms != null ? `${summary.avg_ms} ms` : '—'}
              {slowRate != null ? ` · ${slowRate}% slow` : ''}
            </div>
          </section>

          <section style={styles.railCard}>
            <div style={styles.railHeader}>
              <span style={styles.railLabel}>Latency Alerts</span>
            </div>
            {attention.length === 0 ? (
              <div style={{ ...styles.railEmpty, color: colors.green }}>All targets within SLA</div>
            ) : (
              <ul style={styles.railList}>
                {attention.slice(0, 6).map(s => (
                  <li key={s.service_id}>
                    <Link to={`/performance/${s.service_id}`} style={styles.alertRow}>
                      <span style={{ ...styles.alertDot, background: healthColors[s.health] }} />
                      <div style={{ minWidth: 0 }}>
                        <div style={styles.alertTitle}>{s.name}</div>
                        <div style={styles.alertMeta}>
                          P95 {s.p95_ms} ms · SLA {s.slow_threshold_ms} ms
                        </div>
                      </div>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section style={styles.railCard}>
            <div style={styles.railLabel}>Fleet Avg</div>
            <div style={styles.railValue}>
              {summary?.avg_ms != null ? `${summary.avg_ms} ms` : '—'}
            </div>
            <div style={styles.railMeta}>
              {summary?.total_checks ?? 0} checks in selected period
            </div>
          </section>
        </aside>
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { maxWidth: '100%' },
  layout: {
    display: 'flex',
    gap: 24,
    alignItems: 'flex-start',
    flexWrap: 'wrap',
  },
  main: { flex: '1 1 560px', minWidth: 0 },
  topBar: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 28,
    gap: 16,
    flexWrap: 'wrap',
  },
  metrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))',
    gap: 12,
    marginBottom: 24,
  },
  metricWide: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    padding: 24,
    gridColumn: 'span 2',
  },
  metricCard: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    padding: 24,
  },
  metricStat: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    padding: '20px 16px',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'center',
    gap: 4,
  },
  metricLabel: { fontSize: 12, fontWeight: 500, color: colors.textMuted, marginBottom: 8 },
  metricValue: { fontSize: 28, fontWeight: 700, letterSpacing: '-0.02em', lineHeight: 1.1 },
  metricSub: { fontSize: 12, color: colors.textMuted, marginTop: 8 },
  statCount: { fontSize: 24, fontWeight: 700 },
  statLabel: { fontSize: 12, color: colors.textMuted, fontWeight: 500 },
  toolbar: {
    display: 'flex',
    gap: 12,
    alignItems: 'center',
    marginBottom: 16,
    flexWrap: 'wrap',
  },
  searchInput: { maxWidth: 280, width: '100%' },
  tabs: {
    display: 'flex',
    gap: 4,
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    padding: 4,
  },
  tab: {
    border: 'none',
    background: 'transparent',
    color: colors.textMuted,
    fontSize: 13,
    fontWeight: 500,
    padding: '8px 14px',
    borderRadius: 8,
    display: 'inline-flex',
    alignItems: 'center',
    gap: 6,
    minHeight: 36,
  },
  tabActive: { background: colors.bgElevated, color: colors.text },
  tabCount: { fontSize: 11, color: colors.textDim },
  tableWrap: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    overflow: 'auto',
  },
  table: { width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed', fontSize: 14 },
  th: {
    textAlign: 'left',
    padding: '12px 16px',
    borderBottom: `1px solid ${colors.border}`,
    color: colors.textMuted,
    fontWeight: 600,
    fontSize: 12,
    textTransform: 'uppercase',
    letterSpacing: '0.04em',
    whiteSpace: 'nowrap',
  },
  td: {
    textAlign: 'left',
    padding: '14px 16px',
    borderBottom: `1px solid ${colors.border}`,
    verticalAlign: 'middle',
  },
  tr: {},
  targetLink: { display: 'flex', flexDirection: 'column', gap: 2, textDecoration: 'none', color: colors.text, minWidth: 0 },
  targetName: { fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  targetUrl: { fontSize: 12, color: colors.textMuted, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  healthBadge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 6,
    padding: '4px 10px',
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 600,
  },
  healthDot: { width: 7, height: 7, borderRadius: '50%', flexShrink: 0 },
  kebab: {
    width: 32,
    height: 32,
    borderRadius: 8,
    border: 'none',
    background: 'transparent',
    color: colors.textMuted,
    fontSize: 18,
    lineHeight: 1,
  },
  menu: {
    position: 'absolute',
    right: 0,
    top: '100%',
    zIndex: 20,
    minWidth: 140,
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    borderRadius: 10,
    overflow: 'hidden',
    boxShadow: '0 12px 32px rgba(0,0,0,0.4)',
  },
  menuItem: {
    display: 'block',
    padding: '10px 14px',
    fontSize: 13,
    color: colors.text,
    textDecoration: 'none',
  },
  menuDangerBtn: {
    display: 'block',
    width: '100%',
    textAlign: 'left',
    padding: '10px 14px',
    fontSize: 13,
    color: colors.red,
    border: 'none',
    background: 'transparent',
  },
  empty: {
    textAlign: 'center',
    padding: 48,
    color: colors.textMuted,
    background: colors.card,
    borderRadius: 12,
    border: `1px solid ${colors.border}`,
  },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
  rail: {
    width: 300,
    flex: '0 0 300px',
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
  },
  railCard: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    padding: '18px 20px',
  },
  railHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  railLabel: {
    fontSize: 12,
    fontWeight: 600,
    color: colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: '0.04em',
  },
  railValue: { fontSize: 28, fontWeight: 700, marginTop: 8, letterSpacing: '-0.02em' },
  railMeta: { fontSize: 12, color: colors.textMuted, marginTop: 8 },
  railEmpty: { fontSize: 13, color: colors.textMuted, padding: '20px 0' },
  railList: { listStyle: 'none', margin: '8px 0 0', padding: 0 },
  alertRow: {
    display: 'flex',
    gap: 10,
    alignItems: 'flex-start',
    padding: '10px 0',
    color: colors.text,
    textDecoration: 'none',
    borderBottom: `1px solid ${colors.border}`,
  },
  alertDot: { width: 8, height: 8, borderRadius: '50%', marginTop: 5, flexShrink: 0 },
  alertTitle: { fontWeight: 600, fontSize: 13 },
  alertMeta: { fontSize: 12, color: colors.textMuted, marginTop: 2 },
}
