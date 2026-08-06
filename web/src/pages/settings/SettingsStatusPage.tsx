import { FormEvent, useEffect, useState } from 'react'
import { api, Monitor, StatusPageConfig } from '../../api'
import { colors } from '../../theme'

export default function SettingsStatusPage() {
  const [cfg, setCfg] = useState<StatusPageConfig>({ enabled: false, title: 'System Status', monitor_ids: [] })
  const [monitors, setMonitors] = useState<Monitor[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([api.getStatusPageConfig(), api.monitors()])
      .then(([c, m]) => { setCfg(c); setMonitors(m) })
      .catch(() => {})
  }, [])

  function toggleMonitor(id: string) {
    setCfg(c => ({
      ...c,
      monitor_ids: c.monitor_ids.includes(id)
        ? c.monitor_ids.filter(x => x !== id)
        : [...c.monitor_ids, id],
    }))
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    try {
      const saved = await api.putStatusPageConfig(cfg)
      setCfg(saved)
      setMessage('Status page settings saved')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    }
  }

  return (
    <>
      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}
      <form onSubmit={handleSave} style={styles.card}>
        <h3 style={styles.title}>Public Status Page</h3>
        <p style={styles.desc}>Available at <code>/status</code> when enabled. No login required.</p>
        <label style={styles.check}>
          <input type="checkbox" checked={cfg.enabled} onChange={e => setCfg(c => ({ ...c, enabled: e.target.checked }))} />
          Enable public status page
        </label>
        <label className="field" style={{ marginTop: 16 }}><span className="field-label">Page title</span>
          <input className="input" value={cfg.title} onChange={e => setCfg(c => ({ ...c, title: e.target.value }))} /></label>
        <div style={{ marginTop: 20 }}>
          <div className="field-label" style={{ marginBottom: 12 }}>Monitors to display</div>
          <div style={styles.list}>
            {monitors.map(m => (
              <label key={m.id} style={styles.item}>
                <input type="checkbox" checked={cfg.monitor_ids.includes(m.id)} onChange={() => toggleMonitor(m.id)} />
                {m.name}
              </label>
            ))}
          </div>
        </div>
        <button type="submit" className="btn btn-primary" style={{ marginTop: 20 }}>Save</button>
      </form>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: 28, maxWidth: 640 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 14, margin: '0 0 20px' },
  check: { display: 'flex', gap: 8, alignItems: 'center', fontSize: 14 },
  list: { display: 'grid', gap: 8, maxHeight: 280, overflow: 'auto' },
  item: { display: 'flex', gap: 10, alignItems: 'center', fontSize: 14 },
  ok: { background: 'rgba(34,197,94,0.15)', color: colors.green, padding: 12, borderRadius: 8, marginBottom: 16 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
