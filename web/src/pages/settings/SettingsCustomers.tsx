import { FormEvent, useEffect, useState } from 'react'
import { api, Customer } from '../../api'
import { colors } from '../../theme'

export default function SettingsCustomers() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [name, setName] = useState('')
  const [quota, setQuota] = useState(1)
  const [editing, setEditing] = useState<Customer | null>(null)
  const [editName, setEditName] = useState('')
  const [editQuota, setEditQuota] = useState(1)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function load() {
    try {
      setCustomers(await api.listCustomers())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load customers')
    }
  }

  useEffect(() => { load() }, [])

  async function handleAdd(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    try {
      await api.createCustomer({ name, monitor_quota: quota })
      setName('')
      setQuota(1)
      setMessage('Customer created')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create customer')
    }
  }

  function startEdit(c: Customer) {
    setEditing(c)
    setEditName(c.name)
    setEditQuota(c.monitor_quota)
  }

  async function handleSaveEdit(e: FormEvent) {
    e.preventDefault()
    if (!editing) return
    setError('')
    setMessage('')
    try {
      await api.updateCustomer(editing.id, { name: editName, monitor_quota: editQuota })
      setEditing(null)
      setMessage('Customer updated')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed')
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Delete this customer? Monitors will be unassigned, not deleted.')) return
    setError('')
    setMessage('')
    try {
      await api.deleteCustomer(id)
      setMessage('Customer deleted')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  return (
    <>
      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}

      <div style={styles.grid}>
        <div style={styles.card}>
          <h3 style={styles.cardTitle}>Customers</h3>
          <p style={{ color: colors.textMuted, fontSize: 14, margin: '0 0 20px' }}>
            Group monitors by customer and set monitor quotas (default 1).
          </p>
          {customers.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No customers yet.</p>
          ) : (
            <div style={styles.table}>
              <div style={styles.tableHead}>
                <span>Name</span>
                <span>Usage</span>
                <span></span>
              </div>
              {customers.map(c => (
                <div key={c.id} style={styles.tableRow}>
                  {editing?.id === c.id ? (
                    <form onSubmit={handleSaveEdit} style={styles.editRow}>
                      <input className="input" value={editName} onChange={e => setEditName(e.target.value)} required />
                      <input
                        className="input"
                        type="number"
                        min={1}
                        value={editQuota}
                        onChange={e => setEditQuota(+e.target.value)}
                        style={{ width: 80 }}
                      />
                      <div style={{ display: 'flex', gap: 8 }}>
                        <button type="submit" className="btn btn-primary" style={{ fontSize: 12, padding: '4px 10px' }}>Save</button>
                        <button type="button" className="btn" style={{ fontSize: 12, padding: '4px 10px' }} onClick={() => setEditing(null)}>Cancel</button>
                      </div>
                    </form>
                  ) : (
                    <>
                      <span style={{ fontWeight: 600 }}>{c.name}</span>
                      <span style={{ color: colors.textMuted, fontSize: 13 }}>
                        {c.monitor_count ?? 0} / {c.monitor_quota}
                      </span>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <button type="button" className="btn" style={{ fontSize: 12, padding: '4px 10px' }} onClick={() => startEdit(c)}>Edit</button>
                        <button type="button" className="btn" style={{ fontSize: 12, padding: '4px 10px', color: colors.red }} onClick={() => handleDelete(c.id)}>Delete</button>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <form onSubmit={handleAdd} style={styles.card}>
          <h3 style={styles.cardTitle}>Add Customer</h3>
          <label className="field" style={{ marginBottom: 16 }}>
            <span className="field-label">Name</span>
            <input className="input" value={name} onChange={e => setName(e.target.value)} required />
          </label>
          <label className="field" style={{ marginBottom: 16 }}>
            <span className="field-label">Monitor quota</span>
            <input className="input" type="number" min={1} value={quota} onChange={e => setQuota(+e.target.value)} />
          </label>
          <button type="submit" className="btn btn-primary" style={{ marginTop: 8 }}>Add Customer</button>
        </form>
      </div>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: 20, maxWidth: 960 },
  card: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '24px 28px',
  },
  cardTitle: { margin: '0 0 20px', fontSize: 16, fontWeight: 600 },
  table: { display: 'flex', flexDirection: 'column', gap: 8 },
  tableHead: {
    display: 'grid', gridTemplateColumns: '1fr 100px 140px', gap: 12,
    fontSize: 11, fontWeight: 600, color: colors.textMuted,
    textTransform: 'uppercase', letterSpacing: '0.04em', padding: '0 4px 8px',
  },
  tableRow: {
    display: 'grid', gridTemplateColumns: '1fr 100px 140px', gap: 12, alignItems: 'center',
    padding: '12px 4px', borderTop: `1px solid ${colors.border}`,
  },
  editRow: {
    display: 'grid', gridTemplateColumns: '1fr 80px auto', gap: 12, alignItems: 'center',
    gridColumn: '1 / -1',
  },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(63,185,80,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(248,81,73,0.3)`,
  },
}
