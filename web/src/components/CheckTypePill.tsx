import { colors, radius } from '../theme'

function labelFor(type?: string, url?: string): string {
  if (!type || type === 'http') {
    if (url?.startsWith('https://')) return 'HTTPS'
    if (url?.startsWith('http://')) return 'HTTP'
    return 'HTTPS'
  }
  const labels: Record<string, string> = { port: 'PORT', ssl: 'SSL', dns: 'DNS', heartbeat: 'HEARTBEAT' }
  return labels[type] || type.toUpperCase()
}

function pillColors(type?: string, url?: string): { bg: string; text: string; border: string } {
  const t = type || 'http'
  if (t === 'http') {
    return {
      bg: colors.greenDim,
      text: colors.green,
      border: `color-mix(in srgb, ${colors.green} 40%, transparent)`,
    }
  }
  if (t === 'ssl' || t === 'port' || t === 'dns') {
    return {
      bg: colors.blueDim,
      text: colors.blue,
      border: `color-mix(in srgb, ${colors.blue} 35%, transparent)`,
    }
  }
  return {
    bg: colors.brandDim,
    text: colors.brand,
    border: `color-mix(in srgb, ${colors.brand} 32%, transparent)`,
  }
}

export default function CheckTypePill({ type, url }: { type?: string; url?: string }) {
  const c = pillColors(type, url)
  return (
    <span
      style={{
        background: c.bg,
        color: c.text,
        padding: '3px 9px',
        borderRadius: radius.sm,
        fontSize: 11,
        fontWeight: 700,
        letterSpacing: '0.04em',
        border: `1px solid ${c.border}`,
        whiteSpace: 'nowrap',
      }}
    >
      {labelFor(type, url)}
    </span>
  )
}
