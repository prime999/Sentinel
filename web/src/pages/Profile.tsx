import { FormEvent, useEffect, useState } from 'react'
import { api } from '../api'
import { roleLabel, useAuth } from '../context/AuthContext'
import { colors } from '../theme'

export default function Profile() {
  const { refresh } = useAuth()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    api.getProfile().then(p => {
      setUsername(p.username)
      setEmail(p.email || '')
      setRole(p.role)
    }).catch(() => {})
  }, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')

    if (newPassword && newPassword !== confirmPassword) {
      setError('New passwords do not match')
      return
    }

    try {
      const updated = await api.updateProfile({
        current_password: currentPassword,
        username: username.trim(),
        email: email.trim(),
        new_password: newPassword || undefined,
      })
      setUsername(updated.username)
      setEmail(updated.email || '')
      setRole(updated.role)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMessage('Profile updated successfully')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed')
    }
  }

  const initials = username.slice(0, 2).toUpperCase()

  return (
    <div className="page">
      <h1 className="page-title">Profile</h1>
      <p className="page-subtitle">Manage your account credentials and email for password recovery.</p>

      <div style={styles.headerCard}>
        <span style={styles.avatar}>{initials}</span>
        <div>
          <div style={{ fontWeight: 700, fontSize: 18 }}>{username || 'admin'}</div>
          <div style={{ color: colors.textMuted, fontSize: 14 }}>{role ? roleLabel(role) : 'User'}</div>
        </div>
      </div>

      {message && <div style={styles.ok}>{message}</div>}
      {error && <div style={styles.error}>{error}</div>}

      <form onSubmit={handleSubmit} style={styles.form}>
        <h3 style={styles.sectionTitle}>Account Details</h3>
        <label className="field">
          <span className="field-label">Username</span>
          <input className="input" value={username} onChange={e => setUsername(e.target.value)} required />
        </label>
        <label className="field" style={{ marginTop: 16 }}>
          <span className="field-label">Email</span>
          <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="Used for password reset" />
        </label>

        <h3 style={{ ...styles.sectionTitle, marginTop: 28 }}>Change Password</h3>
        <p style={{ color: colors.textMuted, fontSize: 13, margin: '0 0 16px' }}>
          Enter your current password to confirm any changes.
        </p>

        <label className="field">
          <span className="field-label">Current Password</span>
          <input className="input" type="password" value={currentPassword} onChange={e => setCurrentPassword(e.target.value)} required />
        </label>
        <label className="field" style={{ marginTop: 16 }}>
          <span className="field-label">New Password</span>
          <input className="input" type="password" value={newPassword} onChange={e => setNewPassword(e.target.value)} placeholder="Leave blank to keep current" />
        </label>
        <label className="field" style={{ marginTop: 16 }}>
          <span className="field-label">Confirm New Password</span>
          <input className="input" type="password" value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)} placeholder="Leave blank to keep current" />
        </label>

        <button type="submit" className="btn btn-primary" style={{ marginTop: 24 }}>Save Changes</button>
      </form>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  headerCard: {
    display: 'flex', alignItems: 'center', gap: 16, padding: '20px 24px',
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, marginBottom: 24, maxWidth: 480,
  },
  avatar: {
    width: 56, height: 56, borderRadius: '50%',
    background: `linear-gradient(135deg, ${colors.brand}, ${colors.brandDeep})`,
    color: colors.bg, display: 'grid', placeItems: 'center',
    fontSize: 20, fontWeight: 700,
  },
  form: {
    maxWidth: 480, background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '28px 32px',
  },
  sectionTitle: { margin: '0 0 16px', fontSize: 15, fontWeight: 600 },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 12,
    borderRadius: 8, marginBottom: 16, maxWidth: 480,
    border: `1px solid rgba(63,185,80,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 12,
    borderRadius: 8, marginBottom: 16, maxWidth: 480,
    border: `1px solid rgba(248,81,73,0.3)`,
  },
}
