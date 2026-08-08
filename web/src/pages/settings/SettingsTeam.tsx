import { FormEvent, useEffect, useMemo, useState } from 'react'
import { api, Customer, TeamMember, UserRole } from '../../api'
import ConfirmDialog from '../../components/ConfirmDialog'
import { roleLabel, useAuth } from '../../context/AuthContext'
import { colors } from '../../theme'

type RoleFilter = 'all' | UserRole
type CustomerFilterId = 'all' | 'platform' | string

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
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all')
  const [customerFilter, setCustomerFilter] = useState<CustomerFilterId>('all')
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function load() {
    try {
      setMembers(await api.listTeam())
      if (isPlatformAdmin) {
        setCustomers(await api.listCustomers())
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users')
    }
  }

  useEffect(() => { load() }, [isPlatformAdmin])

  function customerName(id?: string) {
    if (!id) return 'Platform'
    return customers.find(c => c.id === id)?.name || id.slice(0, 8)
  }

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return members.filter(m => {
      if (roleFilter !== 'all' && m.role !== roleFilter) return false
      if (isPlatformAdmin) {
        if (customerFilter === 'platform' && m.tenant_id) return false
        if (customerFilter !== 'all' && customerFilter !== 'platform' && m.tenant_id !== customerFilter) {
          return false
        }
      }
      if (!q) return true
      const hay = [m.username, m.email || '', customerName(m.tenant_id)].join(' ').toLowerCase()
      return hay.includes(q)
    })
  }, [members, search, roleFilter, customerFilter, isPlatformAdmin, customers])

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
      setMessage('User added')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add user')
    }
  }

  async function handleRoleChange(id: string, newRole: UserRole) {
    setError('')
    setMessage('')
    try {
      await api.updateTeamMember(id, { role: newRole })
      setMessage('User updated')
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
      setMessage('User updated')
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
      await api.deleteTeamMember(deleteId)
      setDeleteId(null)
      setMessage('User removed')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setBusy(false)
    }
  }

  const deleteTarget = members.find(m => m.id === deleteId)

  const adminRoleLabel = isPlatformAdmin ? 'Admin - Full Access' : 'Admin - Customer Access'
  const viewerRoleLabel = 'User - Read Only Access'

  return (
    <div className="page">
      <h1 className="page-title">Users</h1>
      <p className="page-subtitle">
        {isPlatformAdmin
          ? 'Manage platform and customer users.'
          : 'Manage users for your customer account.'}
      </p>

      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}

      <div className="split-panels">
        <div style={styles.card}>
          <h3 style={styles.cardTitle}>All users</h3>
          <p style={{ color: colors.textMuted, fontSize: 14, margin: '0 0 16px' }}>
            {isPlatformAdmin
              ? 'Platform admins have full access. Assign a customer for customer admins and users.'
              : 'Admins can manage this customer’s monitors. Users can view only.'}
          </p>

          {members.length > 0 && (
            <div style={styles.filters}>
              <input
                className="input"
                style={styles.search}
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search username or email…"
                autoComplete="off"
              />
              <select
                className="input"
                style={styles.filterSelect}
                value={roleFilter}
                onChange={e => setRoleFilter(e.target.value as RoleFilter)}
              >
                <option value="all">All roles</option>
                <option value="admin">Admin</option>
                <option value="viewer">User</option>
              </select>
              {isPlatformAdmin && (
                <select
                  className="input"
                  style={styles.filterSelect}
                  value={customerFilter}
                  onChange={e => setCustomerFilter(e.target.value)}
                >
                  <option value="all">All customers</option>
                  <option value="platform">Platform</option>
                  {customers.map(c => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
              )}
            </div>
          )}

          {members.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No users yet.</p>
          ) : filtered.length === 0 ? (
            <p style={{ color: colors.textMuted }}>No users match the current filters.</p>
          ) : (
            <div style={styles.tableWrap}>
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>Username</th>
                    <th style={styles.th}>Role</th>
                    {isPlatformAdmin && <th style={styles.th}>Customer</th>}
                    <th style={{ ...styles.th, width: 96, textAlign: 'right' }}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(m => (
                    <tr key={m.id}>
                      <td style={styles.td}>
                        <span style={styles.username}>
                          {m.username}
                          {m.id === currentUser?.id && <span style={styles.youBadge}>You</span>}
                        </span>
                      </td>
                      <td style={styles.td}>
                        <select
                          className="input input-compact"
                          value={m.role}
                          disabled={m.id === currentUser?.id}
                          onChange={e => handleRoleChange(m.id, e.target.value as UserRole)}
                        >
                          <option value="admin">{adminRoleLabel}</option>
                          <option value="viewer">{viewerRoleLabel}</option>
                        </select>
                      </td>
                      {isPlatformAdmin && (
                        <td style={styles.td}>
                          <select
                            className="input input-compact"
                            value={m.tenant_id || ''}
                            disabled={m.id === currentUser?.id}
                            onChange={e => handleTenantChange(m.id, e.target.value)}
                          >
                            <option value="">Platform</option>
                            {customers.map(c => (
                              <option key={c.id} value={c.id}>{c.name}</option>
                            ))}
                          </select>
                        </td>
                      )}
                      <td style={{ ...styles.td, textAlign: 'right' }}>
                        {m.id !== currentUser?.id ? (
                          <button type="button" className="btn" onClick={() => setDeleteId(m.id)} style={styles.deleteBtn}>
                            Remove
                          </button>
                        ) : (
                          <span style={styles.roleBadge}>{roleLabel(m.role)}</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {members.length > 0 && (
            <p style={{ color: colors.textDim, fontSize: 12, margin: '14px 0 0' }}>
              Showing {filtered.length} of {members.length} users
            </p>
          )}
        </div>

        <form
          onSubmit={handleAdd}
          style={styles.card}
          autoComplete="off"
          data-lpignore="true"
          data-1p-ignore="true"
          data-bwignore="true"
        >
          {/* Decoy fields: stop browsers from treating this as a login form */}
          <input type="text" name="username" autoComplete="username" tabIndex={-1} aria-hidden style={styles.honeypot} />
          <input type="password" name="password" autoComplete="current-password" tabIndex={-1} aria-hidden style={styles.honeypot} />

          <h3 style={styles.cardTitle}>Add user</h3>
          <Field label="Username" htmlFor="user-username">
            <CleanInput
              id="user-username"
              name="sentinel_username_new"
              value={username}
              onChange={setUsername}
              required
              autoComplete="off"
            />
          </Field>
          <Field label="Email (optional)" htmlFor="user-email">
            <CleanInput
              id="user-email"
              type="email"
              name="sentinel_email_new"
              value={email}
              onChange={setEmail}
              placeholder="For password reset"
              autoComplete="off"
            />
          </Field>
          <Field label="Password" htmlFor="user-password">
            <CleanInput
              id="user-password"
              type="password"
              name="sentinel_password_new"
              value={password}
              onChange={setPassword}
              required
              minLength={8}
              autoComplete="new-password"
            />
            <p style={{ color: colors.textMuted, fontSize: 12, margin: '8px 0 0' }}>
              At least 8 characters, with a letter and a number.
            </p>
          </Field>
          <Field label="Role" htmlFor="user-role">
            <select id="user-role" className="input" value={role} onChange={e => setRole(e.target.value as UserRole)}>
              <option value="admin">{adminRoleLabel}</option>
              <option value="viewer">{viewerRoleLabel}</option>
            </select>
          </Field>
          {isPlatformAdmin && (
            <Field label="Customer" htmlFor="user-customer">
              <select id="user-customer" className="input" value={tenantId} onChange={e => setTenantId(e.target.value)}>
                <option value="">Platform (no customer)</option>
                {customers.map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            </Field>
          )}
          <button type="submit" className="btn btn-primary" style={{ marginTop: 8 }}>Add user</button>
        </form>
      </div>

      <ConfirmDialog
        open={!!deleteId}
        title="Remove user?"
        message={
          deleteTarget
            ? `Remove “${deleteTarget.username}”? They will lose access immediately.`
            : 'Remove this user? They will lose access immediately.'
        }
        confirmLabel="Remove"
        danger
        busy={busy}
        onConfirm={confirmDelete}
        onCancel={() => { if (!busy) setDeleteId(null) }}
      />
    </div>
  )
}

/** Clipped shell + readonly-until-focus to avoid misaligned browser password UI. */
function CleanInput({
  id,
  name,
  value,
  onChange,
  type = 'text',
  placeholder,
  required,
  minLength,
  autoComplete = 'off',
}: {
  id: string
  name: string
  value: string
  onChange: (v: string) => void
  type?: string
  placeholder?: string
  required?: boolean
  minLength?: number
  autoComplete?: string
}) {
  const [unlocked, setUnlocked] = useState(false)
  return (
    <div className="input-shell">
      <input
        id={id}
        className="input-shell-control"
        type={type}
        name={name}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        required={required}
        minLength={minLength}
        autoComplete={autoComplete}
        autoCorrect="off"
        spellCheck={false}
        readOnly={!unlocked}
        onFocus={() => setUnlocked(true)}
        data-lpignore="true"
        data-1p-ignore="true"
        data-bwignore="true"
        data-form-type="other"
      />
    </div>
  )
}

function Field({
  label,
  htmlFor,
  children,
}: {
  label: string
  htmlFor: string
  children: React.ReactNode
}) {
  return (
    <div className="field" style={{ marginBottom: 18 }}>
      <label className="field-label" htmlFor={htmlFor}>{label}</label>
      {children}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '24px 28px', minWidth: 0,
  },
  cardTitle: { margin: '0 0 20px', fontSize: 16, fontWeight: 600 },
  filters: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 10,
    marginBottom: 16,
  },
  search: {
    flex: '1 1 200px',
    minWidth: 180,
  },
  filterSelect: {
    flex: '1 1 140px',
    minWidth: 140,
    maxWidth: 200,
  },
  tableWrap: { overflowX: 'auto' },
  table: { width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed' },
  th: {
    textAlign: 'left',
    fontSize: 11,
    fontWeight: 600,
    color: colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: '0.04em',
    padding: '0 8px 10px',
    borderBottom: `1px solid ${colors.border}`,
  },
  td: {
    padding: '12px 8px',
    borderBottom: `1px solid ${colors.border}`,
    verticalAlign: 'middle',
  },
  username: { fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 8, minWidth: 0 },
  youBadge: {
    fontSize: 10, fontWeight: 600, color: colors.brand,
    background: 'rgba(20,184,166,0.15)', padding: '2px 6px', borderRadius: 4,
  },
  roleBadge: { fontSize: 12, color: colors.textMuted },
  deleteBtn: { fontSize: 12, padding: '6px 10px', minHeight: 32, color: colors.red },
  honeypot: {
    position: 'absolute',
    left: -9999,
    width: 1,
    height: 1,
    opacity: 0,
    pointerEvents: 'none',
  },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(34,197,94,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, border: `1px solid rgba(239,68,68,0.3)`,
  },
}
