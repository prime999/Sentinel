import { NavLink, Outlet, Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { colors } from '../../theme'

const allTabs = [
  { path: '/settings/general', label: 'General', platformOnly: true },
  { path: '/settings/notifications', label: 'Notifications', platformOnly: false, adminOnly: true },
  { path: '/settings/maintenance', label: 'Maintenance', platformOnly: true },
  { path: '/settings/server', label: 'Server', platformOnly: true },
  { path: '/settings/status-page', label: 'Status Page', platformOnly: true },
  { path: '/settings/tokens', label: 'API Tokens', platformOnly: false },
  { path: '/settings/audit', label: 'Audit', platformOnly: true },
]

export default function SettingsLayout() {
  const location = useLocation()
  const { isPlatformAdmin, isAdmin } = useAuth()
  const tabs = allTabs.filter(t => {
    if (t.platformOnly && !isPlatformAdmin) return false
    if (t.adminOnly && !isAdmin) return false
    return true
  })
  const defaultPath = isPlatformAdmin ? '/settings/general' : (isAdmin ? '/settings/notifications' : '/settings/tokens')

  if (location.pathname === '/settings') {
    return <Navigate to={defaultPath} replace />
  }

  // Legacy redirects
  if (location.pathname === '/settings/smtp') {
    return <Navigate to="/settings/notifications/email" replace />
  }
  if (location.pathname === '/settings/webhooks') {
    return <Navigate to="/settings/notifications/webhooks" replace />
  }

  const allowed = tabs.some(t => location.pathname === t.path || location.pathname.startsWith(t.path + '/'))
  if (!allowed) {
    return <Navigate to={defaultPath} replace />
  }

  return (
    <div className="page">
      <h1 className="page-title">Settings</h1>
      <p className="page-subtitle">
        {isPlatformAdmin
          ? 'Configure organization details and alert delivery.'
          : isAdmin
            ? 'Manage notifications and API tokens for your account.'
            : 'Manage API tokens for this account.'}
      </p>

      <nav style={styles.tabs}>
        {tabs.map(tab => (
          <NavLink
            key={tab.path}
            to={tab.path}
            end={tab.path !== '/settings/notifications'}
            style={({ isActive }) => ({
              ...styles.tab,
              ...(isActive || (tab.path === '/settings/notifications' && location.pathname.startsWith('/settings/notifications'))
                ? styles.tabActive
                : {}),
            })}
          >
            {tab.label}
          </NavLink>
        ))}
      </nav>

      <Outlet />
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  tabs: {
    display: 'flex',
    gap: 4,
    marginBottom: 24,
    padding: 4,
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 10,
    width: 'fit-content',
    flexWrap: 'wrap',
  },
  tab: {
    padding: '8px 18px',
    borderRadius: 8,
    fontSize: 13,
    fontWeight: 500,
    color: colors.textMuted,
    textDecoration: 'none',
    transition: 'background 0.15s, color 0.15s',
  },
  tabActive: {
    background: colors.bgElevated,
    color: colors.text,
  },
}
