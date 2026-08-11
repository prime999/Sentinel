import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Incident, Monitor, MonitorStats } from '../api'
import CustomerFilter, { matchesCustomerFilter } from '../components/CustomerFilter'
import DashboardRail from '../components/DashboardRail'
import DeleteMonitorButton from '../components/DeleteMonitorButton'
import Sparkline from '../components/Sparkline'
import StatusBadge, { badgeStatusFor } from '../components/StatusBadge'
import TypeBadge from '../components/TypeBadge'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'

type StatusTab = 'all' | 'up' | 'degraded' | 'down'

type RowStats = {
  uptime_pct: number
  points: number[]
}

function greetingFor(hour: number): string {
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

function greetingName(name?: string, username?: string): string {
  const n = (name || '').trim()
  if (n) {
    // Prefer first name for the greeting.
    return n.split(/\s+/)[0]
  }
  if (!username) return 'there'
  if (username.includes('.') || username.includes('_') || username.includes(' ')) {
    return username.replace(/[._]/g, ' ').replace(/\b\w/g, c => c.toUpperCase()).split(/\s+/)[0]
  }
  return username.charAt(0).toUpperCase() + username.slice(1)
}

function timeAgo(iso: string): string {
  const sec = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`
  return new Date(iso).toLocaleDateString()
}

function monitorTarget(m: Monitor): string {
  if (m.type === 'heartbeat') return 'Heartbeat monitor'
  if (m.type === 'port') return `${m.url}:${m.port}`
  return m.url
}

export default function Monitors() {
  const { user, isAdmin, isPlatformAdmin } = useAuth()
  const [monitors, setMonitors] = useState<Monitor[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [statsMap, setStatsMap] = useState<Record<string, RowStats>>({})
  const [tagFilter, setTagFilter] = useState('')
  const [statusTab, setStatusTab] = useState<StatusTab>('all')
  const [selectedCustomers, setSelectedCustomers] = useState<string[]>([])
  const [customers, setCustomers] = useState<{ id: string; name: string }[]>([])
  const [search, setSearch] = useState('')
  const [error, setError] = useState('')
  const [menuId, setMenuId] = useState<string | null>(null)
  const menuRef = useRef<HTMLDivElement | null>(null)

  async function load() {
    try {
      const [mons, incs] = await Promise.all([
        api.monitors({ tag: tagFilter || undefined }),
        api.incidents({ limit: 50, offset: 0 }),
      ])
      setMonitors(mons)
      setIncidents(incs.items)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    }
  }

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.listCustomers().then(c => setCustomers(c.map(x => ({ id: x.id, name: x.name })))).catch(() => {})
  }, [isPlatformAdmin])

  useEffect(() => {
    load()
    const id = setInterval(load, 30000)
    return () => clearInterval(id)
  }, [tagFilter])

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!menuRef.current?.contains(e.target as Node)) setMenuId(null)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [])

  const customerScoped = useMemo(
    () => monitors.filter(m => matchesCustomerFilter(m.tenant_id, selectedCustomers)),
    [monitors, selectedCustomers],
  )

  const q = search.trim().toLowerCase()
  const searched = useMemo(() => {
    if (!q) return customerScoped
    return customerScoped.filter(m => {
      const hay = [m.name, m.url, m.type, m.port != null ? String(m.port) : '', ...(m.tags || [])].join(' ').toLowerCase()
      return hay.includes(q)
    })
  }, [customerScoped, q])

  const filtered = useMemo(() => {
    if (statusTab === 'all') return searched
    return searched.filter(m => m.last_status === statusTab)
  }, [searched, statusTab])

  const up = searched.filter(m => m.last_status === 'up').length
  const down = searched.filter(m => m.last_status === 'down').length
  const degraded = searched.filter(m => m.last_status === 'degraded').length

  useEffect(() => {
    let cancelled = false
    const ids = filtered.slice(0, 40).map(m => m.id)
    if (ids.length === 0) {
      setStatsMap({})
      return
    }
    ;(async () => {
      const entries = await Promise.all(
        ids.map(async id => {
          try {
            const s: MonitorStats = await api.stats(id, '30d')
            const points = (s.points || []).slice(-24).map(p => p.response_time_ms)
            return [id, { uptime_pct: s.uptime_pct, points }] as const
          } catch {
            return [id, { uptime_pct: 0, points: [] }] as const
          }
        }),
      )
      if (!cancelled) {
        const next: Record<string, RowStats> = {}
        for (const [id, row] of entries) next[id] = row
        setStatsMap(next)
      }
    })()
    return () => { cancelled = true }
  }, [filtered.map(m => m.id).join(',')])

  const overallUptime = useMemo(() => {
    const vals = filtered.map(m => statsMap[m.id]?.uptime_pct).filter((v): v is number => typeof v === 'number')
    if (vals.length === 0) return null
    return vals.reduce((a, b) => a + b, 0) / vals.length
  }, [filtered, statsMap])

  const allTags = [...new Set(monitors.flatMap(m => m.tags || []))].sort()
  const hour = new Date().getHours()
  const name = greetingName(user?.name, user?.username)

  const statusTabs: { id: StatusTab; label: string; count?: number }[] = [
    { id: 'all', label: 'All', count: searched.length },
    { id: 'up', label: 'Up', count: up },
    { id: 'degraded', label: 'Warning', count: degraded },
    { id: 'down', label: 'Down', count: down },
  ]

  return (
    <div className="page" style={styles.page}>
      <div style={styles.layout}>
        <div style={styles.main}>
          <div style={styles.topBar}>
            <div>
              <h1 className="page-title" style={{ fontSize: 28 }}>
                {greetingFor(hour)}, {name}
              </h1>
              <p className="page-subtitle" style={{ marginBottom: 0 }}>
                Here&apos;s what&apos;s happening with your monitors.
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
              {isAdmin && (
                <Link to="/monitors/new" className="btn btn-primary">+ Add Monitor</Link>
              )}
            </div>
          </div>

          {searched.length > 0 && (
            <div style={styles.metrics}>
              <div style={styles.metricWide}>
                <div style={styles.metricLabel}>Overall Uptime</div>
                <div style={styles.metricValue}>
                  {overallUptime == null ? '—' : `${overallUptime.toFixed(2)}%`}
                </div>
                <div style={styles.metricSub}>Last 30 days · filtered monitors</div>
              </div>
              <div style={styles.metricCard}>
                <div style={styles.metricLabel}>Monitors</div>
                <div style={styles.metricValue}>{searched.length}</div>
                <div style={styles.metricSub}>Total monitors</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(34,197,94,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.green }}>{up}</div>
                <div style={styles.statLabel}>Healthy</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(245,158,11,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.yellow }}>{degraded}</div>
                <div style={styles.statLabel}>Warning</div>
              </div>
              <div style={{ ...styles.metricStat, borderColor: 'rgba(239,68,68,0.35)' }}>
                <div style={{ ...styles.statCount, color: colors.red }}>{down}</div>
                <div style={styles.statLabel}>Down</div>
              </div>
            </div>
          )}

          {(monitors.length > 0 || search) && (
            <div style={styles.toolbar}>
              <input
                className="input"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search monitors…"
                style={styles.searchInput}
              />
              <div style={styles.tabs}>
                {statusTabs.map(tab => (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => setStatusTab(tab.id)}
                    style={{
                      ...styles.tab,
                      ...(statusTab === tab.id ? styles.tabActive : {}),
                    }}
                  >
                    {tab.label}
                    {typeof tab.count === 'number' && (
                      <span style={styles.tabCount}>{tab.count}</span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {allTags.length > 0 && (
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
              <button
                type="button"
                className="btn"
                style={{ fontSize: 13, minHeight: 36, ...(tagFilter === '' ? { background: colors.bgElevated } : {}) }}
                onClick={() => setTagFilter('')}
              >
                All tags
              </button>
              {allTags.map(tag => (
                <button
                  key={tag}
                  type="button"
                  className="btn"
                  style={{ fontSize: 13, minHeight: 36, ...(tagFilter === tag ? { background: colors.bgElevated } : {}) }}
                  onClick={() => setTagFilter(tag)}
                >
                  {tag}
                </button>
              ))}
            </div>
          )}

          {error && <div style={styles.error}>{error}</div>}

          {monitors.length === 0 ? (
            <div style={styles.empty}>
              <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 8 }}>No monitors yet</div>
              <div style={{ color: colors.textMuted, marginBottom: 20 }}>
                Add your first website, port, SSL, or DNS monitor.
              </div>
              {isAdmin && <Link to="/monitors/new" className="btn btn-primary">Add Monitor</Link>}
            </div>
          ) : filtered.length === 0 ? (
            <div style={styles.empty}>
              <div style={{ fontWeight: 600, marginBottom: 8 }}>No matches</div>
              <div style={{ color: colors.textMuted }}>
                {search.trim()
                  ? `No monitors match “${search.trim()}”.`
                  : 'No monitors for the selected filters.'}
              </div>
            </div>
          ) : (
            <div style={styles.tableWrap}>
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>Monitor</th>
                    <th style={styles.th}>Type</th>
                    <th style={styles.th}>Status</th>
                    <th style={styles.th}>Response Time</th>
                    <th style={styles.th}>Uptime (30d)</th>
                    <th style={styles.th}>Last Checked</th>
                    <th style={{ ...styles.th, width: 48 }} />
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(m => {
                    const st = statsMap[m.id]
                    const ms = m.latest_response_time_ms
                    const sparkColor = m.last_status === 'down'
                      ? colors.red
                      : m.last_status === 'degraded'
                        ? colors.yellow
                        : colors.green
                    return (
                      <tr key={m.id} style={styles.tr}>
                        <td style={styles.td}>
                          <Link to={`/monitors/${m.id}`} style={styles.monitorLink}>
                            <span style={styles.monitorName}>{m.name}</span>
                            <span style={styles.monitorUrl}>{monitorTarget(m)}</span>
                          </Link>
                        </td>
                        <td style={styles.td}>
                          <TypeBadge type={m.type} url={m.url} />
                        </td>
                        <td style={styles.td}>
                          <StatusBadge status={badgeStatusFor(m.type, m.last_status)} />
                        </td>
                        <td style={styles.td}>
                          <div style={styles.responseCell}>
                            <span style={{ fontWeight: 600 }}>
                              {typeof ms === 'number' ? `${ms}ms` : '—'}
                            </span>
                            {st?.points && st.points.length > 1 && (
                              <Sparkline values={st.points} color={sparkColor} />
                            )}
                          </div>
                        </td>
                        <td style={styles.td}>
                          {st ? `${st.uptime_pct.toFixed(2)}%` : '—'}
                        </td>
                        <td style={{ ...styles.td, color: colors.textMuted }}>
                          {m.last_checked_at ? timeAgo(m.last_checked_at) : 'Waiting'}
                        </td>
                        <td style={styles.td}>
                          <div style={{ position: 'relative' }} ref={menuId === m.id ? menuRef : undefined}>
                            <button
                              type="button"
                              aria-label="Actions"
                              style={styles.kebab}
                              onClick={() => setMenuId(menuId === m.id ? null : m.id)}
                            >
                              ⋮
                            </button>
                            {menuId === m.id && (
                              <div style={styles.menu}>
                                <Link
                                  to={`/monitors/${m.id}`}
                                  style={styles.menuItem}
                                  onClick={() => setMenuId(null)}
                                >
                                  View
                                </Link>
                                {isAdmin && (
                                  <>
                                    <Link
                                      to={`/monitors/${m.id}/edit`}
                                      style={styles.menuItem}
                                      onClick={() => setMenuId(null)}
                                    >
                                      Edit
                                    </Link>
                                    <div style={styles.menuDanger}>
                                      <DeleteMonitorButton
                                        id={m.id}
                                        name={m.name}
                                        onDeleted={() => {
                                          setMenuId(null)
                                          setMonitors(prev => prev.filter(x => x.id !== m.id))
                                        }}
                                      />
                                    </div>
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

        <DashboardRail incidents={incidents} uptimePct={overallUptime} />
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
  metricLabel: {
    fontSize: 12,
    fontWeight: 500,
    color: colors.textMuted,
    marginBottom: 8,
  },
  metricValue: {
    fontSize: 28,
    fontWeight: 700,
    letterSpacing: '-0.02em',
    lineHeight: 1.1,
  },
  metricSub: {
    fontSize: 12,
    color: colors.textMuted,
    marginTop: 8,
  },
  statCount: {
    fontSize: 24,
    fontWeight: 700,
  },
  statLabel: {
    fontSize: 12,
    color: colors.textMuted,
    fontWeight: 500,
  },
  toolbar: {
    display: 'flex',
    gap: 12,
    alignItems: 'center',
    marginBottom: 16,
    flexWrap: 'wrap',
  },
  searchInput: {
    maxWidth: 280,
    width: '100%',
  },
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
  tabActive: {
    background: colors.bgElevated,
    color: colors.text,
  },
  tabCount: {
    fontSize: 11,
    color: colors.textDim,
  },
  tableWrap: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    overflow: 'auto',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    fontSize: 14,
  },
  th: {
    textAlign: 'left',
    padding: '14px 16px',
    fontSize: 12,
    fontWeight: 600,
    color: colors.textMuted,
    borderBottom: `1px solid ${colors.border}`,
    whiteSpace: 'nowrap',
  },
  tr: {
    borderBottom: `1px solid ${colors.border}`,
  },
  td: {
    padding: '16px',
    verticalAlign: 'middle',
  },
  monitorLink: {
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
    color: 'inherit',
    textDecoration: 'none',
    minWidth: 0,
  },
  monitorName: {
    fontWeight: 600,
  },
  monitorUrl: {
    fontSize: 12,
    color: colors.textMuted,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    maxWidth: 280,
  },
  responseCell: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
  },
  kebab: {
    width: 36,
    height: 36,
    borderRadius: 10,
    border: `1px solid transparent`,
    background: 'transparent',
    color: colors.textMuted,
    fontSize: 18,
    lineHeight: 1,
  },
  menu: {
    position: 'absolute',
    right: 0,
    top: '100%',
    marginTop: 4,
    minWidth: 140,
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    boxShadow: '0 12px 32px rgba(0,0,0,0.45)',
    zIndex: 30,
    overflow: 'hidden',
  },
  menuItem: {
    display: 'block',
    padding: '10px 14px',
    fontSize: 13,
    fontWeight: 500,
    color: colors.text,
    textDecoration: 'none',
    borderBottom: `1px solid ${colors.border}`,
  },
  menuDanger: {
    padding: 8,
  },
  empty: {
    textAlign: 'center',
    padding: '64px 24px',
    background: colors.card,
    borderRadius: colors.radius,
    border: `1px solid ${colors.border}`,
  },
  error: {
    background: colors.redDim,
    color: colors.red,
    padding: 12,
    borderRadius: 10,
    marginBottom: 16,
    border: `1px solid rgba(239,68,68,0.3)`,
  },
}
