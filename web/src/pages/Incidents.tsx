import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Incident, Monitor } from '../api'
import IncidentFilters, { IncidentFilterValues } from '../components/IncidentFilters'
import StatusBadge from '../components/StatusBadge'
import { colors } from '../theme'

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
  const [error, setError] = useState('')

  useEffect(() => {
    api.monitors().then(setMonitors).catch(() => {})
  }, [])

  async function load() {
    try {
      setError('')
      setIncidents(await api.incidents({
        date: filters.date || undefined,
        status: filters.status || undefined,
        type: filters.type || undefined,
        monitorId: filters.monitorId || undefined,
        limit: 200,
      }))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    }
  }

  useEffect(() => {
    load()
    const id = setInterval(load, 30000)
    return () => clearInterval(id)
  }, [filters.date, filters.status, filters.type, filters.monitorId])

  const open = incidents.filter(i => !i.resolved_at).length
  const monitorOptions = monitors.map(m => ({ id: m.id, name: m.name }))

  return (
    <div className="page">
      <div style={styles.topBar}>
        <div>
          <h1 className="page-title">Incidents</h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            {open} open · {incidents.length} shown
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

      {incidents.length === 0 ? (
        <div style={styles.empty}>
          {filters.date || filters.status || filters.type || filters.monitorId
            ? 'No incidents match these filters.'
            : 'No incidents recorded yet.'}
        </div>
      ) : (
        <div style={styles.tableWrap}>
          <table style={styles.table}>
            <thead>
              <tr>
                <th>Monitor</th>
                <th>Type</th>
                <th>Message</th>
                <th>Started</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {incidents.map(inc => (
                <tr key={inc.id}>
                  <td>
                    <Link to={`/monitors/${inc.monitor_id}`} style={styles.link}>
                      {inc.monitor_name || inc.monitor_id}
                    </Link>
                  </td>
                  <td><span style={styles.type}>{inc.type}</span></td>
                  <td style={{ color: colors.textMuted, maxWidth: 320 }}>{inc.message || '—'}</td>
                  <td>{new Date(inc.started_at).toLocaleString()}</td>
                  <td>
                    {inc.resolved_at ? (
                      <span style={styles.resolved}>Resolved</span>
                    ) : (
                      <StatusBadge status="down" />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
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
  tableWrap: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, overflow: 'auto' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 14 },
  link: { color: colors.brand, textDecoration: 'none', fontWeight: 500 },
  type: { textTransform: 'uppercase', fontSize: 11, fontWeight: 700, color: colors.textMuted },
  resolved: { color: colors.green, fontSize: 13, fontWeight: 600 },
  empty: { textAlign: 'center', padding: 48, color: colors.textMuted, background: colors.card, borderRadius: 12, border: `1px solid ${colors.border}` },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
