import { NavLink, Outlet, Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { colors } from '../../theme'

const allTabs = [
  { path: '/settings/general', label: 'General', platformOnly: true },
  { path: '/settings/customers', label: 'Customers', platformOnly: true },
  { path: '/settings/team', label: 'Team', platformOnly: false },
  { path: '/settings/smtp', label: 'SMTP', platformOnly: true },
  { path: '/settings/webhooks', label: 'Webhooks', platformOnly: true },
  { path: '/settings/maintenance', label: 'Maintenance', platformOnly: true },
  { path: '/settings/server', label: 'Server', platformOnly: true },
  { path: '/settings/status-page', label: 'Status Page', platformOnly: true },
  { path: '/settings/tokens', label: 'API Tokens', platformOnly: false },
  { path: '/settings/audit', label: 'Audit', platformOnly: true },
]

export default function SettingsLayout() {
  const location = useLocation()
  const { isPlatformAdmin } = useAuth()
  const tabs = allTabs.filter(t => !t.platformOnly || isPlatformAdmin)
  const defaultPath = isPlatformAdmin ? '/settings/general' : '/settings/team'

  if (location.pathname === '/settings') {
    return <Navigate to={defaultPath} replace />
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
          : 'Manage your team and API tokens.'}
      </p>

      <nav style={styles.tabs}>
        {tabs.map(tab => (
          <NavLink
            key={tab.path}
            to={tab.path}
            style={({ isActive }) => ({
              ...styles.tab,
              ...(isActive ? styles.tabActive : {}),
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
