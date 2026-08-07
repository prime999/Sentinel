import { FormEvent, useEffect, useState } from 'react'
import { api, ServerSettings } from '../../api'
import { colors } from '../../theme'

export default function SettingsServer() {
  const [cfg, setCfg] = useState<ServerSettings>({ dashboard_url: '', retention_days: 30, workers: 10 })
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => { api.getServerSettings().then(setCfg).catch(() => {}) }, [])

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    try {
      const saved = await api.putServerSettings(cfg)
      setCfg(saved)
      setMessage('Server settings saved (restart required for worker/retention changes)')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    }
  }

  return (
    <>
      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}
      <form onSubmit={handleSave} style={styles.card}>
        <h3 style={styles.title}>Server Settings</h3>
        <p style={styles.desc}>Stored in the database. Some values require a process restart to take effect.</p>
        <label className="field"><span className="field-label">Dashboard URL</span>
          <input className="input" value={cfg.dashboard_url} onChange={e => setCfg(c => ({ ...c, dashboard_url: e.target.value }))} /></label>
        <div className="grid-2" style={{ gap: 16 }}>
          <label className="field"><span className="field-label">Retention (days)</span>
            <input
              type="number"
              min={30}
              className="input"
              value={cfg.retention_days}
              onChange={e => setCfg(c => ({ ...c, retention_days: Math.max(30, +e.target.value || 30) }))}
            />
            <span style={{ fontSize: 12, color: colors.textMuted, marginTop: 6 }}>Minimum 30 days — check history is kept for this period.</span>
          </label>
          <label className="field"><span className="field-label">Workers</span>
            <input type="number" className="input" value={cfg.workers} onChange={e => setCfg(c => ({ ...c, workers: +e.target.value }))} /></label>
        </div>
        <button type="submit" className="btn btn-primary" style={{ marginTop: 16 }}>Save</button>
      </form>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: 28, maxWidth: 640 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 14, margin: '0 0 20px' },
  ok: { background: 'rgba(34,197,94,0.15)', color: colors.green, padding: 12, borderRadius: 8, marginBottom: 16 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
