import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { api, Profile } from '../api'
import { roleLabel, useAuth } from '../context/AuthContext'
import { colors } from '../theme'

function initialsFrom(name: string, username: string): string {
  const source = (name || username || 'U').trim()
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return source.slice(0, 2).toUpperCase()
}

export default function ProfileMenu({ onLogout }: { onLogout: () => void }) {
  const { user } = useAuth()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [open, setOpen] = useState(false)
  const location = useLocation()

  useEffect(() => {
    api.getProfile().then(setProfile).catch(() => {})
  }, [location.pathname])

  const username = profile?.username || user?.username || 'admin'
  const name = (profile?.name || user?.name || '').trim()
  const displayName = name || username
  const initials = initialsFrom(name, username)

  return (
    <div style={styles.wrap}>
      <button
        type="button"
        onClick={() => setOpen(v => !v)}
        style={styles.profileBtn}
        aria-expanded={open}
      >
        <span style={styles.avatar}>{initials}</span>
        <span style={styles.profileInfo}>
          <span style={styles.profileName}>{displayName}</span>
          <span style={styles.roleChip}>{profile ? roleLabel(profile.role) : 'Admin'}</span>
        </span>
        <span style={{ ...styles.chevron, transform: open ? 'rotate(180deg)' : 'none' }}>▾</span>
      </button>

      {open && (
        <div style={styles.menu}>
          <div style={styles.signedIn}>Signed in as @{username}</div>
          <Link to="/profile" style={styles.menuItem} onClick={() => setOpen(false)}>
            Profile Settings
          </Link>
          <button type="button" onClick={() => { setOpen(false); onLogout() }} style={styles.menuItemBtn}>
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  wrap: { position: 'relative', padding: '0 4px' },
  profileBtn: {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    width: '100%',
    padding: '10px 12px',
    borderRadius: 12,
    border: 'none',
    background: 'transparent',
    color: colors.text,
    textAlign: 'left',
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: '50%',
    flexShrink: 0,
    background: colors.brandDim,
    color: colors.brand,
    display: 'grid',
    placeItems: 'center',
    fontSize: 13,
    fontWeight: 700,
    border: `1px solid rgba(20, 184, 166, 0.35)`,
  },
  profileInfo: { display: 'flex', flexDirection: 'column', flex: 1, minWidth: 0 },
  profileName: {
    fontSize: 14,
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  roleChip: {
    display: 'inline-flex',
    alignSelf: 'flex-start',
    marginTop: 4,
    padding: '1px 7px',
    borderRadius: 999,
    fontSize: 11,
    fontWeight: 600,
    letterSpacing: '0.02em',
    color: colors.textMuted,
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    lineHeight: 1.4,
  },
  chevron: { fontSize: 10, color: colors.textMuted, transition: 'transform 0.15s', flexShrink: 0 },
  menu: {
    position: 'absolute',
    bottom: '100%',
    left: 4,
    right: 4,
    marginBottom: 8,
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    overflow: 'hidden',
    boxShadow: '0 12px 32px rgba(0,0,0,0.45)',
    zIndex: 20,
  },
  signedIn: {
    padding: '10px 14px 8px',
    fontSize: 12,
    color: colors.textMuted,
    borderBottom: `1px solid ${colors.border}`,
  },
  menuItem: {
    display: 'block',
    padding: '12px 14px',
    fontSize: 13,
    fontWeight: 500,
    color: colors.text,
    textDecoration: 'none',
    borderBottom: `1px solid ${colors.border}`,
  },
  menuItemBtn: {
    display: 'block',
    width: '100%',
    padding: '12px 14px',
    fontSize: 13,
    fontWeight: 500,
    color: colors.red,
    background: 'transparent',
    border: 'none',
    textAlign: 'left',
  },
}
