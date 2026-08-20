import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Incident, Monitor } from '../api'
import { ColGroup, ResizableTh, useColumnResize } from '../components/ColumnResize'
import IncidentFilters, { IncidentFilterValues } from '../components/IncidentFilters'
import IncidentStatus from '../components/IncidentStatus'
import { colors } from '../theme'

const PAGE_SIZE = 20

const emptyFilters: IncidentFilterValues = {
  date: '',
  status: '',
  type: '',
  monitorId: '',
}

export default function Incidents() {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [monitors, setMonitors] = useState<Monitor[]>([])
  const [filters, setFilters] = useState<IncidentFilterValues>(emptyFilters)
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [openCount, setOpenCount] = useState(0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const tableRef = useRef<HTMLTableElement>(null)
  const { widths, startResize, autoFit } = useColumnResize('incidents', 6)

  useEffect(() => {
    api.monitors().then(setMonitors).catch(() => {})
  }, [])

  useEffect(() => {
    setPage(0)
  }, [filters.date, filters.status, filters.type, filters.monitorId])

  async function load() {
    try {
      setError('')
      setLoading(true)
      const [pageRes, openRes] = await Promise.all([
        api.incidents({
          date: filters.date || undefined,
          status: filters.status || undefined,
          type: filters.type || undefined,
          monitorId: filters.monitorId || undefined,
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
        }),
        api.incidents({
          status: 'open',
          monitorId: filters.monitorId || undefined,
          limit: 1,
          offset: 0,
        }),
      ])
      setIncidents(pageRes.items)
      setTotal(pageRes.total)
      setOpenCount(openRes.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const id = setInterval(load, 30000)
    return () => clearInterval(id)
  }, [filters.date, filters.status, filters.type, filters.monitorId, page])

  const monitorOptions = monitors.map(m => ({ id: m.id, name: m.name }))
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1
  const to = Math.min(total, (page + 1) * PAGE_SIZE)

  return (
    <div className="page">
      <div style={styles.topBar}>
        <div>
          <h1 className="page-title">Incidents</h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            {openCount} open · {total} total
            {total > 0 ? ` · Showing ${from}–${to}` : ''}
          </p>
        </div>
      </div>

      <div style={styles.filterCard}>
        <IncidentFilters
          value={filters}
          onChange={setFilters}
          monitors={monitorOptions}
          showMonitor
        />
      </div>

      {error && <div style={styles.error}>{error}</div>}

      {incidents.length === 0 && !loading ? (
        <div style={styles.empty}>
          {filters.date || filters.status || filters.type || filters.monitorId
            ? 'No incidents match these filters.'
            : 'No incidents recorded yet.'}
        </div>
      ) : (
        <div style={styles.tableWrap}>
          <div className="data-table-wrap">
          <table ref={tableRef} className="data-table" style={styles.table}>
            <ColGroup widths={widths} />
            <thead>
              <tr>
                <ResizableTh index={0} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Monitor</ResizableTh>
                <ResizableTh index={1} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Type</ResizableTh>
                <ResizableTh index={2} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Message</ResizableTh>
                <ResizableTh index={3} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Started</ResizableTh>
                <ResizableTh index={4} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Resolved</ResizableTh>
                <ResizableTh index={5} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Status</ResizableTh>
              </tr>
            </thead>
            <tbody>
              {loading && incidents.length === 0 ? (
                <tr>
                  <td colSpan={6} style={{ ...styles.td, padding: 24, color: colors.textMuted }}>Loading…</td>
                </tr>
              ) : (
                incidents.map(inc => (
                  <tr key={inc.id}>
                    <td style={styles.td}>
                      <Link to={`/monitors/${inc.monitor_id}`} style={styles.link}>
                        {inc.monitor_name || inc.monitor_id}
                      </Link>
                    </td>
                    <td style={styles.td}><span style={styles.type}>{inc.type}</span></td>
                    <td style={{ ...styles.td, color: colors.textMuted }}>{inc.message || '—'}</td>
                    <td style={styles.td}>{new Date(inc.started_at).toLocaleString()}</td>
                    <td style={{ ...styles.td, color: colors.textMuted }}>
                      {inc.resolved_at ? new Date(inc.resolved_at).toLocaleString() : '—'}
                    </td>
                    <td style={styles.td}>
                      <IncidentStatus incident={inc} />
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
          </div>
          {total > PAGE_SIZE && (
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
      )}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  topBar: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16, gap: 16, flexWrap: 'wrap' },
  filterCard: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    padding: '14px 16px',
    marginBottom: 20,
  },
  tableWrap: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12 },
  table: { width: '100%', fontSize: 14 },
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
    padding: '12px 16px',
    borderBottom: `1px solid ${colors.border}`,
    verticalAlign: 'middle',
  },
  link: { color: colors.brand, textDecoration: 'none', fontWeight: 500 },
  type: { textTransform: 'uppercase', fontSize: 11, fontWeight: 700, color: colors.textMuted },
  empty: { textAlign: 'center', padding: 48, color: colors.textMuted, background: colors.card, borderRadius: 12, border: `1px solid ${colors.border}` },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
  pager: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    padding: '12px 16px',
    borderTop: `1px solid ${colors.border}`,
  },
  pagerBtn: { padding: '8px 14px', fontSize: 13 },
}
