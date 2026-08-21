import { Link, useLocation } from 'react-router-dom'
import { ReactNode, useEffect, useState } from 'react'
import AppLogo from './AppLogo'
import NavIcon from './NavIcon'
import ProfileMenu from './ProfileMenu'
import { api, OrgSettings } from '../api'
import { useAuth } from '../context/AuthContext'
import { iconSizes, icons, NavIconKey } from '../icons'
import { colors } from '../theme'

const SIDEBAR_WIDTH = 240
const SIDEBAR_COLLAPSED_WIDTH = 72
const COLLAPSE_KEY = 'sentinel.sidebar.collapsed'

function CollapseIcon({ collapsed }: { collapsed: boolean }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path
        d={collapsed ? 'M6 3.5 11 8l-5 4.5' : 'M10 3.5 5 8l5 4.5'}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

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
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem(COLLAPSE_KEY) === '1'
    } catch {
      return false
    }
  })

  function toggleCollapsed() {
    setCollapsed(v => {
      const next = !v
      try {
        localStorage.setItem(COLLAPSE_KEY, next ? '1' : '0')
      } catch {
        /* ignore quota */
      }
      return next
    })
  }

  const navItems = allNavItems.filter(item => {
    if (item.platformOnly && !isPlatformAdmin) return false
    if (item.adminOnly && !isAdmin) return false
    return true
  })

  useEffect(() => {
    api.getGeneral().then(setOrg).catch(() => {})
  }, [location.pathname])

  const brandName = org?.company_name || 'Sentinel'
  const tagline = org?.tagline || 'Infrastructure Monitoring'

  return (
    <div style={styles.shell}>
      <aside style={{
        ...styles.sidebar,
        width: collapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH,
        padding: collapsed ? '20px 8px' : '24px 12px',
      }}>
        <div style={{
          ...styles.brandHeader,
          flexDirection: collapsed ? 'column' : 'row',
          alignItems: collapsed ? 'center' : 'flex-start',
        }}>
          <Link to="/" style={{ ...styles.logo, padding: collapsed ? '0 0 4px' : '0 0 0 2px' }} title={brandName}>
          <AppLogo src={org?.logo} size={iconSizes.brandLogo} alt={brandName} />
            {!collapsed && (
              <span style={styles.logoText}>
                <span style={styles.logoName}>{brandName}</span>
                <span style={styles.logoTagline}>{tagline}</span>
              </span>
            )}
          </Link>
          <button
            type="button"
            onClick={toggleCollapsed}
            style={styles.collapseBtn}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <CollapseIcon collapsed={collapsed} />
          </button>
        </div>

        <nav style={styles.nav}>
          {navItems.map(item => {
            const active = isActivePath(location.pathname, item.path)
            return (
              <Link
                key={item.path}
                to={item.path}
                title={item.label}
                style={{
                  ...styles.navItem,
                  ...(active ? styles.navActive : {}),
                  ...(collapsed ? styles.navItemCollapsed : {}),
                }}
              >
                {!collapsed && active && <span style={styles.navAccent} />}
                <NavIcon src={icons[item.icon]} size={iconSizes.nav} alt={item.label} />
                {!collapsed && item.label}
              </Link>
            )
          })}
        </nav>

        <div style={styles.sidebarFooter}>
          <ProfileMenu onLogout={onLogout} collapsed={collapsed} />
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
    width: SIDEBAR_WIDTH,
    flexShrink: 0,
    height: '100%',
    background: colors.sidebar,
    borderRight: `1px solid ${colors.border}`,
    display: 'flex',
    flexDirection: 'column',
    padding: '24px 12px',
    overflow: 'visible',
    zIndex: 4,
    transition: 'width 0.18s ease, padding 0.18s ease',
  },
  brandHeader: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: 4,
    padding: '0 0 20px',
    flexShrink: 0,
  },
  logo: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    flex: 1,
    minWidth: 0,
    color: colors.text,
    textDecoration: 'none',
  },
  collapseBtn: {
    flexShrink: 0,
    width: 28,
    height: 28,
    display: 'grid',
    placeItems: 'center',
    marginTop: 4,
    borderRadius: 8,
    border: `1px solid ${colors.border}`,
    background: colors.bgElevated,
    color: colors.textMuted,
    cursor: 'pointer',
    padding: 0,
  },
  logoText: { display: 'flex', flexDirection: 'column', minWidth: 0, gap: 2 },
  logoName: {
    fontWeight: 700,
    fontSize: 16,
    letterSpacing: '-0.01em',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  logoTagline: {
    fontSize: 11,
    fontWeight: 500,
    color: colors.textMuted,
    lineHeight: 1.3,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
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
  navItemCollapsed: {
    justifyContent: 'center',
    padding: '11px 0',
    gap: 0,
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
