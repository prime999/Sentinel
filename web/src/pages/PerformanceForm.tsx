import { FormEvent, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, Customer, PerformanceTarget } from '../api'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'

const defaults: Partial<PerformanceTarget> = {
  method: 'GET',
  interval_seconds: 300,
  timeout_ms: 10000,
  slow_threshold_ms: 3000,
  follow_redirects: true,
  enabled: true,
  alert_after_slow: 1,
}

export default function PerformanceForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { isPlatformAdmin } = useAuth()
  const [form, setForm] = useState<Partial<PerformanceTarget>>(defaults)
  const [error, setError] = useState('')
  const [customers, setCustomers] = useState<Customer[]>([])

  useEffect(() => {
    if (id) api.getPerformanceTarget(id).then(setForm).catch(() => {})
  }, [id])

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.listCustomers().then(setCustomers).catch(() => {})
  }, [isPlatformAdmin])

  function set<K extends keyof PerformanceTarget>(key: K, value: PerformanceTarget[K]) {
    setForm(prev => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      if (id) {
        await api.updatePerformanceTarget(id, form)
        navigate(`/performance/${id}`)
      } else {
        const created = await api.createPerformanceTarget(form)
        navigate(`/performance/${created.id}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    }
  }

  async function handleDelete() {
    if (!id || !confirm('Delete this performance target?')) return
    try {
      await api.deletePerformanceTarget(id)
      navigate('/performance')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  return (
    <div className="page">
      <h1 className="page-title">{id ? 'Edit Target' : 'Add Performance Target'}</h1>
      <p className="page-subtitle">Track response time and latency for a website — independent of uptime monitoring.</p>
      {error && <div style={styles.error}>{error}</div>}
      <form onSubmit={handleSubmit} style={styles.form}>
        <Field label="Name">
          <input required className="input" value={form.name || ''} onChange={e => set('name', e.target.value)} />
        </Field>
        {isPlatformAdmin && (
          <Field label="Customer">
            <select
              className="input"
              value={form.tenant_id || ''}
              onChange={e => set('tenant_id', e.target.value)}
            >
              <option value="">Internal (unassigned)</option>
              {customers.map(c => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </Field>
        )}
        <Field label="URL">
          <input required type="url" className="input" value={form.url || ''} onChange={e => set('url', e.target.value)} placeholder="https://example.com" />
        </Field>
        <Field label="Method">
          <select className="input" value={form.method || 'GET'} onChange={e => set('method', e.target.value)}>
            <option value="GET">GET</option>
            <option value="HEAD">HEAD</option>
          </select>
        </Field>
        <div className="grid-2" style={{ gap: 16 }}>
          <Field label="Check Interval (seconds)">
            <input type="number" className="input" value={form.interval_seconds ?? 300} onChange={e => set('interval_seconds', +e.target.value)} />
          </Field>
          <Field label="Slow Threshold (ms)">
            <input type="number" className="input" value={form.slow_threshold_ms ?? 3000} onChange={e => set('slow_threshold_ms', +e.target.value)} />
          </Field>
          <Field label="Timeout (ms)">
            <input type="number" className="input" value={form.timeout_ms ?? 10000} onChange={e => set('timeout_ms', +e.target.value)} />
          </Field>
          <Field label="Alert after consecutive slow checks">
            <input
              type="number"
              min={1}
              className="input"
              value={form.alert_after_slow ?? 1}
              onChange={e => set('alert_after_slow', Math.max(1, +e.target.value || 1))}
            />
          </Field>
        </div>
        <p style={{ color: colors.textMuted, fontSize: 13, margin: '-8px 0 16px' }}>
          Send a SLOW alert only after this many slow checks in a row (default 1).
        </p>
        <label style={styles.checkbox}>
          <input type="checkbox" checked={form.follow_redirects ?? true} onChange={e => set('follow_redirects', e.target.checked)} />
          <span>Follow redirects</span>
        </label>
        <Field label="Alert emails (comma-separated)">
          <input className="input" value={form.alert_emails || ''} onChange={e => set('alert_emails', e.target.value)} />
        </Field>
        {id && (
          <label style={{ ...styles.checkbox, marginTop: 12 }}>
            <input type="checkbox" checked={form.enabled ?? true} onChange={e => set('enabled', e.target.checked)} />
            <span>Enabled</span>
          </label>
        )}
        <div style={{ display: 'flex', gap: 10, marginTop: 24 }}>
          <button type="submit" className="btn btn-primary">Save</button>
          {id && (
            <button type="button" className="btn" onClick={handleDelete} style={{ color: colors.red }}>Delete</button>
          )}
        </div>
      </form>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="field" style={{ marginBottom: 16 }}>
      <span className="field-label">{label}</span>
      {children}
    </label>
  )
}

const styles: Record<string, React.CSSProperties> = {
  form: {
    maxWidth: 520, background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '28px 32px',
  },
  checkbox: { display: 'flex', gap: 10, alignItems: 'center', fontSize: 14, color: colors.textMuted },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(248,81,73,0.3)`,
  },
}
