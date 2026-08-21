import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Incident, Monitor } from '../api'
import { ColGroup, ResizableTh, useColumnResize } from '../components/ColumnResize'
import CustomerFilter, { matchesCustomerFilter } from '../components/CustomerFilter'
import DashboardRail from '../components/DashboardRail'
import DeleteMonitorButton from '../components/DeleteMonitorButton'
import MetricCard from '../components/MetricCard'
import PageHeader from '../components/PageHeader'
import Panel from '../components/Panel'
import SegmentedTabs from '../components/SegmentedTabs'
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
  const tableRef = useRef<HTMLTableElement>(null)
  const { widths, startResize, autoFit } = useColumnResize('monitors', 7)

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
      try {
        const next = await api.monitorStatsSummary('30d', ids)
        if (!cancelled) setStatsMap(next)
      } catch {
        if (!cancelled) {
          const empty: Record<string, RowStats> = {}
          for (const id of ids) empty[id] = { uptime_pct: 0, points: [] }
          setStatsMap(empty)
        }
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
      <div className="page-layout">
        <div className="page-layout-main">
          <PageHeader
            title={`${greetingFor(hour)}, ${name}`}
            subtitle="Here's what's happening with your monitors."
            actions={
              <>
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
              </>
            }
          />

          {searched.length > 0 && (
            <div className="kpi-strip">
              <div className="kpi-wide">
                <MetricCard
                  label="Overall Uptime"
                  value={overallUptime == null ? '—' : `${overallUptime.toFixed(2)}%`}
                  sub="Last 30 days · filtered monitors"
                />
              </div>
              <MetricCard label="Monitors" value={String(searched.length)} sub="Total monitors" />
              <MetricCard label="Healthy" value={String(up)} accent="green" />
              <MetricCard label="Warning" value={String(degraded)} accent="yellow" />
              <MetricCard label="Down" value={String(down)} accent="red" />
            </div>
          )}

          {(monitors.length > 0 || search) && (
            <div className="toolbar-row">
              <input
                className="input search-field"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search monitors…"
                aria-label="Search monitors"
              />
              <SegmentedTabs
                label="Filter by status"
                value={statusTab}
                onChange={id => setStatusTab(id as StatusTab)}
                tabs={statusTabs}
              />
            </div>
          )}

          {allTags.length > 0 && (
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }} role="group" aria-label="Filter by tag">
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

          {error && <div className="flash-error" role="alert">{error}</div>}

          {monitors.length === 0 ? (
            <div className="empty-state">
              <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 8, color: colors.text }}>No monitors yet</div>
              <div style={{ marginBottom: 20 }}>
                Add your first website, port, SSL, or DNS monitor.
              </div>
              {isAdmin && <Link to="/monitors/new" className="btn btn-primary">Add Monitor</Link>}
            </div>
          ) : filtered.length === 0 ? (
            <div className="empty-state">
              <div style={{ fontWeight: 600, marginBottom: 8, color: colors.text }}>No matches</div>
              <div>
                {search.trim()
                  ? `No monitors match “${search.trim()}”.`
                  : 'No monitors for the selected filters.'}
              </div>
            </div>
          ) : (
            <Panel padded={false} className="data-table-wrap">
              <table ref={tableRef} className="data-table">
                <ColGroup widths={widths} />
                <thead>
                  <tr>
                    <ResizableTh index={0} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Monitor</ResizableTh>
                    <ResizableTh index={1} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Type</ResizableTh>
                    <ResizableTh index={2} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Status</ResizableTh>
                    <ResizableTh index={3} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Response Time</ResizableTh>
                    <ResizableTh index={4} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Uptime (30d)</ResizableTh>
                    <ResizableTh index={5} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Last Checked</ResizableTh>
                    <ResizableTh index={6} style={{ ...styles.th, width: 48 }} startResize={startResize} autoFit={autoFit} tableRef={tableRef} />
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
                      <tr
                        key={m.id}
                        className={m.last_status === 'down' ? 'row-down' : m.last_status === 'degraded' ? 'row-warn' : undefined}
                      >
                        <td>
                          <Link to={`/monitors/${m.id}`} style={styles.monitorLink}>
                            <span style={styles.monitorName}>{m.name}</span>
                            <span style={styles.monitorUrl}>{monitorTarget(m)}</span>
                          </Link>
                        </td>
                        <td>
                          <TypeBadge type={m.type} url={m.url} />
                        </td>
                        <td>
                          <StatusBadge status={badgeStatusFor(m.type, m.last_status)} />
                        </td>
                        <td>
                          <div style={styles.responseCell}>
                            <span className="num" style={{ fontWeight: 600 }}>
                              {typeof ms === 'number' ? `${ms}ms` : '—'}
                            </span>
                            {st?.points && st.points.length > 1 && (
                              <Sparkline values={st.points} color={sparkColor} />
                            )}
                          </div>
                        </td>
                        <td className="num">
                          {st ? `${st.uptime_pct.toFixed(2)}%` : '—'}
                        </td>
                        <td className="num" style={{ color: colors.textMuted }}>
                          {m.last_checked_at ? timeAgo(m.last_checked_at) : 'Waiting'}
                        </td>
                        <td>
                          <div style={{ position: 'relative' }} ref={menuId === m.id ? menuRef : undefined}>
                            <button
                              type="button"
                              aria-label="Actions"
                              aria-haspopup="menu"
                              aria-expanded={menuId === m.id}
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
            </Panel>
          )}
        </div>

        <DashboardRail incidents={incidents} uptimePct={overallUptime} />
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { maxWidth: '100%' },
  th: {},
  monitorLink: {
    display: 'flex',
    flexDirection: 'column',
    gap: 2,
    color: 'inherit',
    textDecoration: 'none',
    minWidth: 0,
  },
  monitorName: {
    fontWeight: 600,
  },
  monitorUrl: {
    fontSize: 11,
    color: colors.textMuted,
  },
  responseCell: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
  },
  kebab: {
    width: 32,
    height: 32,
    borderRadius: 6,
    border: '1px solid transparent',
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
    borderRadius: 10,
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
}
