import { Link, useLocation } from 'react-router-dom'
import { ReactNode, useEffect, useState } from 'react'
import AppLogo from './AppLogo'
import NavIcon from './NavIcon'
import ProfileMenu from './ProfileMenu'
import { api, OrgSettings } from '../api'
import { useAuth } from '../context/AuthContext'
import { iconSizes, icons, NavIconKey } from '../icons'
import { colors } from '../theme'

type NavItem = {
  path: string
  label: string
  icon: NavIconKey
  adminOnly?: boolean
  platformOnly?: boolean
}

const allNavItems: NavItem[] = [
  { path: '/', label: 'Monitors', icon: 'monitors' },
  { path: '/incidents', label: 'Incidents', icon: 'incidents' },
  { path: '/performance', label: 'Performance', icon: 'performance' },
  { path: '/customers', label: 'Customers', icon: 'customers', platformOnly: true },
  { path: '/users', label: 'Users', icon: 'users', adminOnly: true },
  { path: '/settings', label: 'Settings', icon: 'settings', adminOnly: true },
]

function isActivePath(pathname: string, path: string) {
  if (path === '/') {
    return pathname === '/' || (pathname.startsWith('/monitors/') && !pathname.endsWith('/new'))
  }
  if (path === '/performance') return pathname.startsWith('/performance')
  if (path === '/incidents') return pathname.startsWith('/incidents')
  if (path === '/customers') return pathname.startsWith('/customers')
  if (path === '/users') return pathname.startsWith('/users')
  return pathname.startsWith(path)
}

export default function Layout({ children, onLogout }: { children: ReactNode; onLogout: () => void }) {
  const location = useLocation()
  const { isAdmin, isPlatformAdmin } = useAuth()
  const [org, setOrg] = useState<OrgSettings | null>(null)

  const navItems = allNavItems.filter(item => {
    if (item.platformOnly && !isPlatformAdmin) return false
    if (item.adminOnly && !isAdmin) return false
    return true
  })

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.getGeneral().then(setOrg).catch(() => {})
  }, [location.pathname, isPlatformAdmin])

  const brandName = org?.company_name || 'Sentinel'
  const tagline = org?.tagline || 'Infrastructure Monitoring'

  return (
    <div style={styles.shell}>
      <aside style={styles.sidebar}>
        <Link to="/" style={styles.logo}>
          <AppLogo src={org?.logo} size={iconSizes.brandLogo} />
          <span style={styles.logoText}>
            <span style={styles.logoName}>{brandName}</span>
            <span style={styles.logoTagline}>{tagline}</span>
          </span>
        </Link>

        <nav style={styles.nav}>
          {navItems.map(item => {
            const active = isActivePath(location.pathname, item.path)
            return (
              <Link
                key={item.path}
                to={item.path}
                style={{
                  ...styles.navItem,
                  ...(active ? styles.navActive : {}),
                }}
              >
                {active && <span style={styles.navAccent} />}
                <NavIcon src={icons[item.icon]} size={iconSizes.nav} alt={item.label} />
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
    height: '100vh',
    width: '100%',
    overflow: 'hidden',
    background: colors.bg,
  },
  sidebar: {
    width: 260,
    flexShrink: 0,
    height: '100%',
    background: colors.sidebar,
    borderRight: `1px solid ${colors.border}`,
    display: 'flex',
    flexDirection: 'column',
    padding: '24px 14px',
    overflow: 'hidden',
  },
  logo: {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    padding: '4px 10px 28px',
    color: colors.text,
    textDecoration: 'none',
    flexShrink: 0,
  },
  logoText: { display: 'flex', flexDirection: 'column', minWidth: 0, gap: 2 },
  logoName: { fontWeight: 700, fontSize: 16, letterSpacing: '-0.01em' },
  logoTagline: { fontSize: 11, fontWeight: 500, color: colors.textMuted, lineHeight: 1.3 },
  nav: {
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
    flex: 1,
    minHeight: 0,
    overflowY: 'auto',
  },
  navItem: {
    position: 'relative',
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    padding: '11px 14px',
    borderRadius: 12,
    color: colors.textMuted,
    fontSize: 14,
    fontWeight: 500,
    textDecoration: 'none',
    transition: 'background 0.15s, color 0.15s',
  },
  navActive: {
    background: colors.brandDim,
    color: colors.brand,
  },
  navAccent: {
    position: 'absolute',
    left: 0,
    top: 10,
    bottom: 10,
    width: 3,
    borderRadius: 2,
    background: colors.brand,
  },
  sidebarFooter: {
    flexShrink: 0,
    borderTop: `1px solid ${colors.border}`,
    paddingTop: 16,
    marginTop: 12,
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    minWidth: 0,
    minHeight: 0,
    overflow: 'auto',
    background: colors.bg,
  },
  content: {
    flex: 1,
    padding: '28px 32px',
    width: '100%',
  },
}
