import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, WebhookConfig } from '../../api'
import { useSaveToast } from '../../components/useSaveToast'
import { colors } from '../../theme'

const emptyHook = (): WebhookConfig => ({ url: '', enabled: true, events: ['all'] })

export default function SettingsWebhooks() {
  const [hooks, setHooks] = useState<WebhookConfig[]>([])
  const [error, setError] = useState('')
  const { toast, showSaved } = useSaveToast()

  useEffect(() => { api.getWebhooks().then(h => setHooks(h.length ? h : [emptyHook()])).catch(() => {}) }, [])

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const saved = await api.putWebhooks(hooks.filter(h => h.url.trim()))
      setHooks(saved.length ? saved : [emptyHook()])
      showSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    }
  }

  return (
    <>
      {toast}
      <div style={{ marginBottom: 16 }}>
        <Link to="/settings/notifications" style={{ color: colors.textMuted, fontSize: 14, textDecoration: 'none' }}>
          ← Notifications
        </Link>
      </div>
      {error && <div style={styles.error} role="alert">{error}</div>}
      <form onSubmit={handleSave} style={styles.card}>
        <h3 style={styles.title}>Webhook Notifications</h3>
        <p style={styles.desc}>POST JSON payloads to external URLs on alerts (down, recovery, slow, SSL, DNS).</p>
        {hooks.map((hook, i) => (
          <div key={i} style={styles.row}>
            <input className="input" placeholder="https://hooks.example.com/alert" value={hook.url}
              onChange={e => setHooks(prev => prev.map((h, j) => j === i ? { ...h, url: e.target.value } : h))} />
            <input className="input" placeholder="Events: all, DOWN, RECOVERY" value={hook.events.join(',')}
              onChange={e => setHooks(prev => prev.map((h, j) => j === i ? { ...h, events: e.target.value.split(',').map(s => s.trim()).filter(Boolean) } : h))} />
            <label style={styles.check}>
              <input type="checkbox" checked={hook.enabled} onChange={e => setHooks(prev => prev.map((h, j) => j === i ? { ...h, enabled: e.target.checked } : h))} />
              Enabled
            </label>
            <button type="button" className="btn" onClick={() => setHooks(prev => prev.filter((_, j) => j !== i))}>Remove</button>
          </div>
        ))}
        <div style={{ display: 'flex', gap: 12, marginTop: 16 }}>
          <button type="button" className="btn" onClick={() => setHooks(prev => [...prev, emptyHook()])}>+ Add webhook</button>
          <button type="submit" className="btn btn-primary">Save</button>
        </div>
      </form>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 10, padding: 28, maxWidth: 800 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 15, margin: '0 0 20px' },
  row: { display: 'grid', gap: 10, marginBottom: 16, paddingBottom: 16, borderBottom: `1px solid ${colors.border}` },
  check: { display: 'flex', gap: 8, alignItems: 'center', fontSize: 15 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
