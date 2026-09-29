import { FormEvent, useEffect, useState } from 'react'
import { api, LogSettings } from '../../api'
import { useSaveToast } from '../../components/useSaveToast'
import { colors } from '../../theme'

const defaults: LogSettings = {
  retention_days: 7,
  volume_retention_days: 30,
  max_db_size_bytes: 2 * 1024 * 1024 * 1024,
  max_events_per_sec_host: 100,
  max_events_per_sec_global: 1000,
  max_event_size_bytes: 32 * 1024,
  max_batch_events: 200,
}

export default function SettingsLogs() {
  const [cfg, setCfg] = useState<LogSettings>(defaults)
  const [error, setError] = useState('')
  const { toast, showSaved } = useSaveToast()

  useEffect(() => {
    api.getLogSettings().then(setCfg).catch(() => {})
  }, [])

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const saved = await api.putLogSettings(cfg)
      setCfg(saved)
      showSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    }
  }

  return (
    <>
      {toast}
      {error && <div style={styles.error} role="alert">{error}</div>}
      <form onSubmit={handleSave} style={styles.card}>
        <h3 style={styles.title}>Log collection</h3>
        <p style={styles.desc}>
          Matched log events are stored in a separate <code>sentinel-logs.db</code> file with WAL mode.
          Caps apply on both the agent and the server. Only allowlisted paths (e.g. /var/log, app storage/logs) can be followed.
        </p>
        <div style={styles.stack}>
          <div className="grid-2">
            <label className="field">
              <span className="field-label">Event retention (days)</span>
              <input type="number" min={1} className="input" value={cfg.retention_days}
                onChange={e => setCfg(c => ({ ...c, retention_days: Math.max(1, +e.target.value || 1) }))} />
            </label>
            <label className="field">
              <span className="field-label">Volume bucket retention (days)</span>
              <input type="number" min={1} className="input" value={cfg.volume_retention_days}
                onChange={e => setCfg(c => ({ ...c, volume_retention_days: Math.max(1, +e.target.value || 1) }))} />
            </label>
          </div>
          <label className="field">
            <span className="field-label">Max log database size (MB)</span>
            <input type="number" min={64} className="input"
              value={Math.round(cfg.max_db_size_bytes / (1024 * 1024))}
              onChange={e => setCfg(c => ({ ...c, max_db_size_bytes: Math.max(64, +e.target.value || 64) * 1024 * 1024 }))} />
          </label>
          <div className="grid-2">
            <label className="field">
              <span className="field-label">Max events/sec per host</span>
              <input type="number" min={1} className="input" value={cfg.max_events_per_sec_host}
                onChange={e => setCfg(c => ({ ...c, max_events_per_sec_host: Math.max(1, +e.target.value || 1) }))} />
            </label>
            <label className="field">
              <span className="field-label">Max events/sec global</span>
              <input type="number" min={1} className="input" value={cfg.max_events_per_sec_global}
                onChange={e => setCfg(c => ({ ...c, max_events_per_sec_global: Math.max(1, +e.target.value || 1) }))} />
            </label>
          </div>
          <div className="grid-2">
            <label className="field">
              <span className="field-label">Max event size (KB)</span>
              <input type="number" min={1} className="input"
                value={Math.round(cfg.max_event_size_bytes / 1024)}
                onChange={e => setCfg(c => ({ ...c, max_event_size_bytes: Math.max(1, +e.target.value || 1) * 1024 }))} />
            </label>
            <label className="field">
              <span className="field-label">Max events per ingest batch</span>
              <input type="number" min={10} max={500} className="input" value={cfg.max_batch_events}
                onChange={e => setCfg(c => ({ ...c, max_batch_events: Math.min(500, Math.max(10, +e.target.value || 10)) }))} />
            </label>
          </div>
        </div>
        <div style={styles.actions}>
          <button type="submit" className="btn btn-primary">Save</button>
        </div>
      </form>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 10, padding: '32px' },
  title: { margin: '0 0 8px', fontSize: 17, fontWeight: 600 },
  desc: { color: colors.textMuted, fontSize: 15, margin: '0 0 24px', lineHeight: 1.5 },
  stack: { display: 'flex', flexDirection: 'column', gap: 20 },
  actions: { display: 'flex', justifyContent: 'flex-start', marginTop: 24 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
