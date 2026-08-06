import { Link, useLocation } from 'react-router-dom'
import { ReactNode, useEffect, useState } from 'react'
import AppLogo from './AppLogo'
import ProfileMenu from './ProfileMenu'
import { api, OrgSettings } from '../api'
import { useAuth } from '../context/AuthContext'
import { colors } from '../theme'

const allNavItems = [
  { path: '/', label: 'Monitors', icon: '◉', adminOnly: false },
  { path: '/incidents', label: 'Incidents', icon: '⚡', adminOnly: false },
  { path: '/performance', label: 'Performance', icon: '▤', adminOnly: false },
  { path: '/settings', label: 'Settings', icon: '⚙', adminOnly: true },
]

export default function Layout({ children, onLogout }: { children: ReactNode; onLogout: () => void }) {
  const location = useLocation()
  const { isAdmin, isPlatformAdmin } = useAuth()
  const [org, setOrg] = useState<OrgSettings | null>(null)
  const navItems = allNavItems.filter(item => !item.adminOnly || isAdmin)

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.getGeneral().then(setOrg).catch(() => {})
  }, [location.pathname, isPlatformAdmin])

  const brandName = org?.company_name || 'Sentinel'

  return (
    <div style={styles.shell}>
      <aside style={styles.sidebar}>
        <Link to="/" style={styles.logo}>
          <AppLogo src={org?.logo} size={32} />
          <span style={styles.logoText}>
            <span>{brandName}</span>
            {org?.tagline && <span style={styles.logoTagline}>{org.tagline}</span>}
          </span>
        </Link>

        <nav style={styles.nav}>
          {navItems.map(item => {
            const active = item.path === '/'
              ? location.pathname === '/' || (location.pathname.startsWith('/monitors/') && !location.pathname.endsWith('/new'))
              : item.path === '/performance'
                ? location.pathname.startsWith('/performance')
                : item.path === '/incidents'
                  ? location.pathname.startsWith('/incidents')
                  : location.pathname.startsWith(item.path)
            return (
              <Link
                key={item.path}
                to={item.path}
                style={{ ...styles.navItem, ...(active ? styles.navActive : {}) }}
              >
                <span style={styles.navIcon}>{item.icon}</span>
                {item.label}
              </Link>
            )
          })}
        </nav>

        <div style={styles.sidebarFooter}>
          <ProfileMenu onLogout={onLogout} />
        </div>
      </aside>

      <div style={styles.main}>
        <div style={styles.content}>{children}</div>
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  shell: {
    display: 'flex',
    minHeight: '100vh',
    width: '100%',
    background: colors.bg,
  },
  sidebar: {
    width: 240,
    flexShrink: 0,
    background: colors.sidebar,
    borderRight: `1px solid ${colors.border}`,
    display: 'flex',
    flexDirection: 'column',
    padding: '20px 12px',
  },
  logo: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '8px 12px',
    marginBottom: 24,
    color: colors.text,
    fontWeight: 700,
    fontSize: 18,
    textDecoration: 'none',
  },
  logoText: { display: 'flex', flexDirection: 'column', minWidth: 0 },
  logoTagline: { fontSize: 10, fontWeight: 500, color: colors.textMuted, marginTop: 2, lineHeight: 1.2 },
  nav: { display: 'flex', flexDirection: 'column', gap: 4, flex: 1 },
  navItem: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '10px 12px',
    borderRadius: 8,
    color: colors.textMuted,
    fontSize: 14,
    fontWeight: 500,
    textDecoration: 'none',
    transition: 'background 0.15s',
  },
  navActive: {
    background: colors.card,
    color: colors.text,
  },
  navIcon: { width: 20, textAlign: 'center', fontSize: 14 },
  sidebarFooter: {
    borderTop: `1px solid ${colors.border}`,
    paddingTop: 16,
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    minWidth: 0,
    overflow: 'auto',
  },
  content: {
    flex: 1,
    padding: '24px 32px',
    width: '100%',
  },
}
