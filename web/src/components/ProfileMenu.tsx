import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { api, Profile } from '../api'
import { roleLabel } from '../context/AuthContext'
import { colors } from '../theme'

export default function ProfileMenu({ onLogout }: { onLogout: () => void }) {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [open, setOpen] = useState(false)
  const location = useLocation()

  useEffect(() => {
    api.getProfile().then(setProfile).catch(() => {})
  }, [location.pathname])

  const username = profile?.username || 'admin'
  const initials = username.slice(0, 2).toUpperCase()

  return (
    <div style={styles.wrap}>
      <div style={styles.statusPill}>
        <span style={styles.statusDot} />
        System Online
      </div>

      <button
        type="button"
        onClick={() => setOpen(v => !v)}
        style={styles.profileBtn}
        aria-expanded={open}
      >
        <span style={styles.avatar}>{initials}</span>
        <span style={styles.profileInfo}>
          <span style={styles.profileName}>{username}</span>
          <span style={styles.profileRole}>{profile ? roleLabel(profile.role) : 'Admin'}</span>
        </span>
        <span style={{ ...styles.chevron, transform: open ? 'rotate(180deg)' : 'none' }}>▾</span>
      </button>

      {open && (
        <div style={styles.menu}>
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
  statusPill: {
    display: 'flex', alignItems: 'center', gap: 8,
    fontSize: 11, color: colors.textMuted, padding: '0 8px 12px',
  },
  statusDot: {
    width: 7, height: 7, borderRadius: '50%',
    background: colors.green, boxShadow: `0 0 6px ${colors.green}`,
  },
  profileBtn: {
    display: 'flex', alignItems: 'center', gap: 10, width: '100%',
    padding: '10px 10px', borderRadius: 10, border: `1px solid ${colors.border}`,
    background: colors.card, color: colors.text, textAlign: 'left',
  },
  avatar: {
    width: 36, height: 36, borderRadius: '50%', flexShrink: 0,
    background: `linear-gradient(135deg, ${colors.brand}, ${colors.brandDeep})`,
    color: colors.bg, display: 'grid', placeItems: 'center',
    fontSize: 13, fontWeight: 700,
  },
  profileInfo: { display: 'flex', flexDirection: 'column', flex: 1, minWidth: 0 },
  profileName: {
    fontSize: 13, fontWeight: 600, overflow: 'hidden',
    textOverflow: 'ellipsis', whiteSpace: 'nowrap',
  },
  profileRole: { fontSize: 11, color: colors.textMuted },
  chevron: { fontSize: 10, color: colors.textMuted, transition: 'transform 0.15s' },
  menu: {
    position: 'absolute', bottom: '100%', left: 4, right: 4, marginBottom: 6,
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 10, overflow: 'hidden', boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
  },
  menuItem: {
    display: 'block', padding: '10px 14px', fontSize: 13, fontWeight: 500,
    color: colors.text, textDecoration: 'none',
    borderBottom: `1px solid ${colors.border}`,
  },
  menuItemBtn: {
    display: 'block', width: '100%', padding: '10px 14px', fontSize: 13,
    fontWeight: 500, color: colors.red, background: 'transparent', border: 'none',
    textAlign: 'left',
  },
}
