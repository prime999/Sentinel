import { FormEvent, useEffect, useState } from 'react'
import { api, MaintenanceWindow, Monitor } from '../../api'
import ConfirmDialog from '../../components/ConfirmDialog'
import { colors } from '../../theme'

export default function SettingsMaintenance() {
  const [windows, setWindows] = useState<MaintenanceWindow[]>([])
  const [monitors, setMonitors] = useState<Monitor[]>([])
  const [name, setName] = useState('')
  const [monitorId, setMonitorId] = useState('')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [error, setError] = useState('')
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function load() {
    const [w, m] = await Promise.all([api.listMaintenance(), api.monitors()])
    setWindows(w)
    setMonitors(m)
  }

  useEffect(() => { load().catch(() => {}) }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.createMaintenance({
        name,
        monitor_id: monitorId || undefined,
        starts_at: new Date(startsAt).toISOString(),
        ends_at: new Date(endsAt).toISOString(),
      })
      setName('')
      setMonitorId('')
      setStartsAt('')
      setEndsAt('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    }
  }

  async function confirmDelete() {
    if (!deleteId) return
    setBusy(true)
    setError('')
    try {
      await api.deleteMaintenance(deleteId)
      setDeleteId(null)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setBusy(false)
    }
  }

  const deleteTarget = windows.find(w => w.id === deleteId)

  return (
    <>
      {error && <div style={styles.error}>{error}</div>}
      <form onSubmit={handleCreate} style={styles.card}>
        <h3 style={styles.title}>Schedule Maintenance</h3>
        <p style={styles.desc}>Suppress alerts during planned downtime. Leave monitor blank for global maintenance.</p>
        <div className="grid-2" style={{ gap: 16 }}>
          <label className="field"><span className="field-label">Name</span>
            <input required className="input" value={name} onChange={e => setName(e.target.value)} /></label>
          <label className="field"><span className="field-label">Monitor (optional)</span>
            <select className="input" value={monitorId} onChange={e => setMonitorId(e.target.value)}>
              <option value="">All monitors</option>
              {monitors.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
            </select></label>
          <label className="field"><span className="field-label">Starts</span>
            <input required type="datetime-local" className="input" value={startsAt} onChange={e => setStartsAt(e.target.value)} /></label>
          <label className="field"><span className="field-label">Ends</span>
            <input required type="datetime-local" className="input" value={endsAt} onChange={e => setEndsAt(e.target.value)} /></label>
        </div>
        <button type="submit" className="btn btn-primary" style={{ marginTop: 16 }}>Add window</button>
      </form>

      <div style={{ ...styles.card, marginTop: 24 }}>
        <h3 style={styles.title}>Scheduled Windows</h3>
        {windows.length === 0 ? (
          <p style={styles.desc}>No maintenance windows scheduled.</p>
        ) : (
          <table style={styles.table}>
            <thead><tr><th>Name</th><th>Scope</th><th>Period</th><th></th></tr></thead>
            <tbody>
              {windows.map(w => (
                <tr key={w.id}>
                  <td>{w.name}</td>
                  <td>{w.monitor_id ? monitors.find(m => m.id === w.monitor_id)?.name || w.monitor_id : 'All'}</td>
                  <td>{new Date(w.starts_at).toLocaleString()} – {new Date(w.ends_at).toLocaleString()}</td>
                  <td><button type="button" className="btn" onClick={() => setDeleteId(w.id)}>Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <ConfirmDialog
        open={!!deleteId}
        title="Delete maintenance window?"
        message={
          deleteTarget
            ? `Delete “${deleteTarget.name}”? Alerts will no longer be suppressed for this window.`
            : 'Delete this maintenance window?'
        }
        confirmLabel="Delete"
        danger
        busy={busy}
        onConfirm={confirmDelete}
        onCancel={() => { if (!busy) setDeleteId(null) }}
      />
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: 28 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 14, margin: '0 0 20px' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 14 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
