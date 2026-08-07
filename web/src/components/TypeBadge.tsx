import { colors } from '../theme'

function labelFor(type?: string, url?: string): string {
  if (!type || type === 'http') {
    if (url?.startsWith('https://')) return 'HTTPS'
    if (url?.startsWith('http://')) return 'HTTP'
    return 'HTTP'
  }
  const labels: Record<string, string> = { port: 'PORT', ssl: 'SSL', dns: 'DNS', heartbeat: 'HEARTBEAT' }
  return labels[type] || type.toUpperCase()
}

export default function TypeBadge({ type, url }: { type?: string; url?: string }) {
  return (
    <span style={{
      background: colors.brandDim,
      color: colors.brand,
      padding: '4px 10px',
      borderRadius: 8,
      fontSize: 11,
      fontWeight: 700,
      letterSpacing: '0.06em',
      border: `1px solid rgba(20, 184, 166, 0.3)`,
    }}>
      {labelFor(type, url)}
    </span>
  )
}
