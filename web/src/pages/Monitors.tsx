import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Monitor } from '../api'
import DeleteMonitorButton from '../components/DeleteMonitorButton'
import MetricCard from '../components/MetricCard'
import StatusBadge from '../components/StatusBadge'
import TypeBadge from '../components/TypeBadge'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'

export default function Monitors() {
  const { isAdmin, isPlatformAdmin } = useAuth()
  const [monitors, setMonitors] = useState<Monitor[]>([])
  const [tagFilter, setTagFilter] = useState('')
  const [customerFilter, setCustomerFilter] = useState('')
  const [customers, setCustomers] = useState<{ id: string; name: string }[]>([])
  const [search, setSearch] = useState('')
  const [error, setError] = useState('')

  async function load() {
    try {
      setMonitors(await api.monitors({ tag: tagFilter || undefined, customer: customerFilter || undefined }))
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
  }, [tagFilter, customerFilter])

  const allTags = [...new Set(monitors.flatMap(m => m.tags || []))].sort()
  const q = search.trim().toLowerCase()
  const filtered = q
    ? monitors.filter(m => {
        const hay = [
          m.name,
          m.url,
          m.type,
          m.port != null ? String(m.port) : '',
          ...(m.tags || []),
        ].join(' ').toLowerCase()
        return hay.includes(q)
      })
    : monitors

  const up = filtered.filter(m => m.last_status === 'up').length
  const down = filtered.filter(m => m.last_status === 'down').length
  const degraded = filtered.filter(m => m.last_status === 'degraded').length

  return (
    <div className="page">
      <div style={styles.topBar}>
        <div>
          <h1 className="page-title">Monitors</h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            {filtered.length === monitors.length
              ? `${monitors.length} monitor${monitors.length !== 1 ? 's' : ''} configured`
              : `${filtered.length} of ${monitors.length} monitors`}
          </p>
        </div>
        {isAdmin && <Link to="/monitors/new" className="btn btn-primary">+ Add Monitor</Link>}
      </div>

      {(monitors.length > 0 || search) && (
        <div style={styles.searchRow}>
          <input
            className="input"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search by name, URL, type, or tag…"
            style={styles.searchInput}
          />
        </div>
      )}

      {isPlatformAdmin && customers.length > 0 && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16, alignItems: 'center' }}>
          <span style={{ fontSize: 12, color: colors.textMuted, fontWeight: 600 }}>Customer:</span>
          <button type="button" className="btn" style={{ fontSize: 13, ...(customerFilter === '' ? { background: colors.bgElevated } : {}) }}
            onClick={() => setCustomerFilter('')}>All</button>
          {customers.map(c => (
            <button key={c.id} type="button" className="btn" style={{ fontSize: 13, ...(customerFilter === c.id ? { background: colors.bgElevated } : {}) }}
              onClick={() => setCustomerFilter(c.id)}>{c.name}</button>
          ))}
        </div>
      )}

      {monitors.length > 0 && (
        <div className="grid-4" style={{ marginBottom: 28 }}>
          <MetricCard label="Total" value={String(filtered.length)} accent="blue" />
          <MetricCard label="Online" value={String(up)} accent="green" />
          <MetricCard label="Warning" value={String(degraded)} accent="yellow" />
          <MetricCard label="Offline" value={String(down)} accent="red" />
        </div>
      )}

      {allTags.length > 0 && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 20 }}>
          <button type="button" className="btn" style={{ fontSize: 13, ...(tagFilter === '' ? { background: colors.bgElevated } : {}) }}
            onClick={() => setTagFilter('')}>All</button>
          {allTags.map(tag => (
            <button key={tag} type="button" className="btn" style={{ fontSize: 13, ...(tagFilter === tag ? { background: colors.bgElevated } : {}) }}
              onClick={() => setTagFilter(tag)}>{tag}</button>
          ))}
        </div>
      )}

      {error && <div style={styles.error}>{error}</div>}

      {monitors.length === 0 ? (
        <div style={styles.empty}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>◉</div>
          <div style={{ fontWeight: 600, marginBottom: 8 }}>No monitors yet</div>
          <div style={{ color: colors.textMuted, marginBottom: 20 }}>Add your first website, port, SSL, or DNS monitor.</div>
          {isAdmin && <Link to="/monitors/new" className="btn btn-primary">Add Monitor</Link>}
        </div>
      ) : filtered.length === 0 ? (
        <div style={styles.empty}>
          <div style={{ fontWeight: 600, marginBottom: 8 }}>No matches</div>
          <div style={{ color: colors.textMuted }}>No monitors match “{search.trim()}”.</div>
        </div>
      ) : (
        <div style={styles.grid}>
          {filtered.map(m => (
            <div key={m.id} style={styles.card}>
              <Link to={`/monitors/${m.id}`} style={styles.cardBody}>
                <div style={styles.cardHeader}>
                  <div>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 6 }}>
                      <span style={{ fontWeight: 600, fontSize: 16 }}>{m.name}</span>
                      <TypeBadge type={m.type} url={m.url} />
                    </div>
                    <div style={{ color: colors.textMuted, fontSize: 13 }}>
                      {m.type === 'heartbeat' ? 'Heartbeat monitor' : m.type === 'port' ? `${m.url}:${m.port}` : m.url}
                    </div>
                    {(m.tags?.length ?? 0) > 0 && (
                      <div style={{ display: 'flex', gap: 6, marginTop: 8, flexWrap: 'wrap' }}>
                        {m.tags!.map(tag => (
                          <span key={tag} style={styles.tag}>{tag}</span>
                        ))}
                      </div>
                    )}
                  </div>
                  <StatusBadge status={m.last_status} />
                </div>
                <div style={styles.cardMeta}>
                  {m.last_checked_at ? (
                    <span>Last checked {timeAgo(m.last_checked_at)}</span>
                  ) : (
                    <span>Waiting for first check</span>
                  )}
                </div>
              </Link>
              {isAdmin && (
                <div style={styles.cardActions}>
                  <Link to={`/monitors/${m.id}/edit`} className="btn" style={{ fontSize: 13, padding: '6px 12px' }}>Edit</Link>
                  <DeleteMonitorButton
                    id={m.id}
                    name={m.name}
                    onDeleted={() => setMonitors(prev => prev.filter(x => x.id !== m.id))}
                  />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function timeAgo(iso: string): string {
  const sec = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`
  return new Date(iso).toLocaleString()
}

const styles: Record<string, React.CSSProperties> = {
  topBar: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24,
  },
  searchRow: { marginBottom: 20 },
  searchInput: { maxWidth: 420, width: '100%' },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
    gap: 16,
  },
  card: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    overflow: 'hidden',
    transition: 'border-color 0.15s',
  },
  cardBody: { display: 'block', padding: 20, color: 'inherit', textDecoration: 'none' },
  cardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' },
  cardMeta: { marginTop: 16, fontSize: 13, color: colors.textDim },
  cardActions: {
    display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 16px',
    borderTop: `1px solid ${colors.border}`, background: colors.bgElevated,
  },
  empty: {
    textAlign: 'center', padding: '64px 24px', background: colors.card,
    borderRadius: 12, border: `1px solid ${colors.border}`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(248,81,73,0.3)`,
  },
  tag: {
    fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 4,
    background: colors.bgElevated, color: colors.textMuted, border: `1px solid ${colors.border}`,
  },
}
