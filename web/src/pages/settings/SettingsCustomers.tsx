import { FormEvent, useEffect, useMemo, useState } from 'react'
import { api, Customer } from '../../api'
import ConfirmDialog from '../../components/ConfirmDialog'
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
  const [search, setSearch] = useState('')
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function load() {
    try {
      setCustomers(await api.listCustomers())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load customers')
    }
  }

  useEffect(() => { load() }, [])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return customers
    return customers.filter(c => c.name.toLowerCase().includes(q))
  }, [customers, search])

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

  async function confirmDelete() {
    if (!deleteId) return
    setBusy(true)
    setError('')
    setMessage('')
    try {
      await api.deleteCustomer(deleteId)
      setDeleteId(null)
      setMessage('Customer deleted')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setBusy(false)
    }
  }

  const deleteTarget = customers.find(c => c.id === deleteId)

  return (
    <div className="page">
      <h1 className="page-title">Customers</h1>
      <p className="page-subtitle">Group monitors by customer and set monitor quotas.</p>

      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}

      <div className="split-panels">
        <div style={styles.card}>
          <h3 style={styles.cardTitle}>All customers</h3>
          <p style={{ color: colors.textMuted, fontSize: 14, margin: '0 0 16px' }}>
            Default quota is 1 monitor per customer.
          </p>

          {customers.length > 0 && (
            <input
              className="input"
              style={styles.search}
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search customers…"
              autoComplete="off"
            />
          )}

          {customers.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No customers yet.</p>
          ) : filtered.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No customers match “{search.trim()}”.</p>
          ) : (
            <div style={styles.table}>
              <div style={styles.tableHead}>
                <span>Name</span>
                <span>Usage</span>
                <span>Actions</span>
              </div>
              {filtered.map(c => (
                <div key={c.id} style={styles.tableRow}>
                  {editing?.id === c.id ? (
                    <form onSubmit={handleSaveEdit} style={styles.editRow}>
                      <input className="input input-compact" value={editName} onChange={e => setEditName(e.target.value)} required />
                      <input
                        className="input input-compact"
                        type="number"
                        min={1}
                        value={editQuota}
                        onChange={e => setEditQuota(+e.target.value)}
                        style={{ width: 80 }}
                      />
                      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                        <button type="submit" className="btn btn-primary" style={styles.rowBtn}>Save</button>
                        <button type="button" className="btn" style={styles.rowBtn} onClick={() => setEditing(null)}>Cancel</button>
                      </div>
                    </form>
                  ) : (
                    <>
                      <span style={{ fontWeight: 600, overflowWrap: 'anywhere' }}>{c.name}</span>
                      <span style={{ color: colors.textMuted, fontSize: 13 }}>
                        {c.monitor_count ?? 0} / {c.monitor_quota}
                      </span>
                      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                        <button type="button" className="btn" style={styles.rowBtn} onClick={() => startEdit(c)}>Edit</button>
                        <button type="button" className="btn" style={{ ...styles.rowBtn, color: colors.red }} onClick={() => setDeleteId(c.id)}>Delete</button>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
          {customers.length > 0 && (
            <p style={{ color: colors.textDim, fontSize: 12, margin: '14px 0 0' }}>
              Showing {filtered.length} of {customers.length} customers
            </p>
          )}
        </div>

        <form onSubmit={handleAdd} style={styles.card} autoComplete="off">
          <h3 style={styles.cardTitle}>Add Customer</h3>
          <div className="field" style={{ marginBottom: 16 }}>
            <label className="field-label" htmlFor="customer-name">Name</label>
            <input id="customer-name" className="input" value={name} onChange={e => setName(e.target.value)} required autoComplete="off" />
          </div>
          <div className="field" style={{ marginBottom: 16 }}>
            <label className="field-label" htmlFor="customer-quota">Monitor quota</label>
            <input id="customer-quota" className="input" type="number" min={1} value={quota} onChange={e => setQuota(+e.target.value)} />
          </div>
          <button type="submit" className="btn btn-primary" style={{ marginTop: 8 }}>Add Customer</button>
        </form>
      </div>

      <ConfirmDialog
        open={!!deleteId}
        title="Delete customer?"
        message={
          deleteTarget
            ? `Delete “${deleteTarget.name}”? Monitors will be unassigned, not deleted.`
            : 'Delete this customer? Monitors will be unassigned, not deleted.'
        }
        confirmLabel="Delete"
        danger
        busy={busy}
        onConfirm={confirmDelete}
        onCancel={() => { if (!busy) setDeleteId(null) }}
      />
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '24px 28px', minWidth: 0,
  },
  cardTitle: { margin: '0 0 20px', fontSize: 16, fontWeight: 600 },
  search: { maxWidth: 360, width: '100%', marginBottom: 16 },
  table: { display: 'flex', flexDirection: 'column', gap: 0 },
  tableHead: {
    display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 88px 140px', gap: 12, alignItems: 'center',
    fontSize: 11, fontWeight: 600, color: colors.textMuted,
    textTransform: 'uppercase', letterSpacing: '0.04em', padding: '0 4px 10px',
  },
  tableRow: {
    display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 88px 140px', gap: 12, alignItems: 'center',
    padding: '12px 4px', borderTop: `1px solid ${colors.border}`,
  },
  editRow: {
    display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 80px 140px', gap: 12, alignItems: 'center',
    gridColumn: '1 / -1',
  },
  rowBtn: { fontSize: 12, padding: '6px 10px', minHeight: 32 },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(34,197,94,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(239,68,68,0.3)`,
  },
}
