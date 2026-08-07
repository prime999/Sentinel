import { FormEvent, useEffect, useState } from 'react'
import { api, Customer, TeamMember, UserRole } from '../../api'
import { roleLabel, useAuth } from '../../context/AuthContext'
import { colors } from '../../theme'

export default function SettingsTeam() {
  const { user: currentUser, isPlatformAdmin } = useAuth()
  const [members, setMembers] = useState<TeamMember[]>([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<UserRole>('viewer')
  const [tenantId, setTenantId] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function load() {
    try {
      setMembers(await api.listTeam())
      if (isPlatformAdmin) {
        setCustomers(await api.listCustomers())
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load team')
    }
  }

  useEffect(() => { load() }, [isPlatformAdmin])

  async function handleAdd(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    try {
      await api.createTeamMember({
        username,
        email: email.trim() || undefined,
        password,
        role,
        tenant_id: isPlatformAdmin ? (tenantId || undefined) : undefined,
      })
      setUsername('')
      setEmail('')
      setPassword('')
      setRole('viewer')
      setTenantId('')
      setMessage('Team member added')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add member')
    }
  }

  async function handleRoleChange(id: string, newRole: UserRole) {
    setError('')
    setMessage('')
    try {
      await api.updateTeamMember(id, { role: newRole })
      setMessage('Member updated')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed')
    }
  }

  async function handleTenantChange(id: string, newTenant: string) {
    setError('')
    setMessage('')
    try {
      await api.updateTeamMember(id, { tenant_id: newTenant })
      setMessage('Member updated')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed')
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Remove this team member?')) return
    setError('')
    setMessage('')
    try {
      await api.deleteTeamMember(id)
      setMessage('Member removed')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  function customerName(id?: string) {
    if (!id) return 'Platform'
    return customers.find(c => c.id === id)?.name || id.slice(0, 8)
  }

  const adminRoleLabel = isPlatformAdmin ? 'Admin - Full Access' : 'Admin - Customer Access'
  const viewerRoleLabel = 'User - Read Only Access'

  return (
    <>
      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}

      <div style={styles.grid}>
        <div style={styles.card}>
          <h3 style={styles.cardTitle}>Team Members</h3>
          <p style={{ color: colors.textMuted, fontSize: 14, margin: '0 0 20px' }}>
            {isPlatformAdmin
              ? 'Platform admins have full access. Assign a customer for customer admins and users.'
              : 'Admins can manage this customer’s monitors. Users can view only.'}
          </p>

          {members.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No team members yet.</p>
          ) : (
            <div style={styles.table}>
              <div style={{ ...styles.tableHead, gridTemplateColumns: isPlatformAdmin ? '1fr minmax(180px, auto) minmax(160px, auto) 100px' : '1fr minmax(220px, auto) 100px' }}>
                <span>Username</span>
                <span>Role</span>
                {isPlatformAdmin && <span>Customer</span>}
                <span></span>
              </div>
              {members.map(m => (
                <div key={m.id} style={{ ...styles.tableRow, gridTemplateColumns: isPlatformAdmin ? '1fr minmax(180px, auto) minmax(160px, auto) 100px' : '1fr minmax(220px, auto) 100px' }}>
                  <span style={styles.username}>
                    {m.username}
                    {m.id === currentUser?.id && <span style={styles.youBadge}>You</span>}
                  </span>
                  <select
                    className="input"
                    value={m.role}
                    disabled={m.id === currentUser?.id}
                    onChange={e => handleRoleChange(m.id, e.target.value as UserRole)}
                    style={{ width: '100%', minWidth: 180, padding: '6px 32px 6px 10px', fontSize: 13 }}
                  >
                    <option value="admin">{adminRoleLabel}</option>
                    <option value="viewer">{viewerRoleLabel}</option>
                  </select>
                  {isPlatformAdmin && (
                    <select
                      className="input"
                      value={m.tenant_id || ''}
                      disabled={m.id === currentUser?.id}
                      onChange={e => handleTenantChange(m.id, e.target.value)}
                      style={{ width: '100%', minWidth: 140, padding: '6px 32px 6px 10px', fontSize: 13 }}
                    >
                      <option value="">Platform</option>
                      {customers.map(c => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                    </select>
                  )}
                  <div style={styles.actions}>
                    {m.id !== currentUser?.id ? (
                      <button type="button" className="btn" onClick={() => handleDelete(m.id)} style={styles.deleteBtn}>
                        Remove
                      </button>
                    ) : (
                      <span style={styles.roleBadge}>
                        {roleLabel(m.role)}
                        {isPlatformAdmin && m.tenant_id ? ` · ${customerName(m.tenant_id)}` : ''}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <form onSubmit={handleAdd} style={styles.card}>
          <h3 style={styles.cardTitle}>Add Member</h3>
          <Field label="Username">
            <input className="input" value={username} onChange={e => setUsername(e.target.value)} required />
          </Field>
          <Field label="Email (optional)">
            <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="For password reset" />
          </Field>
          <Field label="Password">
            <input className="input" type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
            <p style={{ color: colors.textMuted, fontSize: 12, margin: '6px 0 0' }}>
              At least 8 characters, with a letter and a number.
            </p>
          </Field>
          <Field label="Role">
            <select className="input" value={role} onChange={e => setRole(e.target.value as UserRole)}>
              <option value="admin">{adminRoleLabel}</option>
              <option value="viewer">{viewerRoleLabel}</option>
            </select>
          </Field>
          {isPlatformAdmin && (
            <Field label="Customer">
              <select className="input" value={tenantId} onChange={e => setTenantId(e.target.value)}>
                <option value="">Platform (no customer)</option>
                {customers.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            </Field>
          )}
          <button type="submit" className="btn btn-primary" style={{ marginTop: 8 }}>Add Member</button>
        </form>
      </div>
    </>
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
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: 20, maxWidth: 1100 },
  card: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '24px 28px',
  },
  cardTitle: { margin: '0 0 20px', fontSize: 16, fontWeight: 600 },
  table: { display: 'flex', flexDirection: 'column', gap: 8 },
  tableHead: {
    display: 'grid', gap: 12,
    fontSize: 11, fontWeight: 600, color: colors.textMuted,
    textTransform: 'uppercase', letterSpacing: '0.04em', padding: '0 4px 8px',
  },
  tableRow: {
    display: 'grid', gap: 12, alignItems: 'center',
    padding: '12px 4px', borderTop: `1px solid ${colors.border}`,
  },
  username: { fontWeight: 600, display: 'flex', alignItems: 'center', gap: 8 },
  youBadge: {
    fontSize: 10, fontWeight: 600, color: colors.brand,
    background: 'rgba(58,175,169,0.15)', padding: '2px 6px', borderRadius: 4,
  },
  roleBadge: { fontSize: 12, color: colors.textMuted },
  actions: { display: 'flex', alignItems: 'center', gap: 8 },
  deleteBtn: { fontSize: 12, padding: '4px 10px', color: colors.red },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(63,185,80,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(248,81,73,0.3)`,
  },
}
