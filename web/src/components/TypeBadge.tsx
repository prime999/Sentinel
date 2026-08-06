import { colors } from '../theme'

function labelFor(type?: string, url?: string): string {
  if (!type || type === 'http') {
    if (url?.startsWith('https://')) return 'HTTPS'
    if (url?.startsWith('http://')) return 'HTTP'
    return 'HTTP/HTTPS'
  }
  const labels: Record<string, string> = { port: 'PORT', ssl: 'SSL', dns: 'DNS', heartbeat: 'HEARTBEAT' }
  return labels[type] || type.toUpperCase()
}

export default function TypeBadge({ type, url }: { type?: string; url?: string }) {
  return (
    <span style={{
      background: 'rgba(58, 175, 169, 0.2)',
      color: colors.brand,
      padding: '3px 10px',
      borderRadius: 6,
      fontSize: 11,
      fontWeight: 700,
      letterSpacing: '0.08em',
      border: `1px solid rgba(58, 175, 169, 0.35)`,
    }}>
      {labelFor(type, url)}
    </span>
  )
}
