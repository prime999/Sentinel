import MonitorKindIcon, { kindColor, monitorKindFor } from './MonitorKindIcon'
import { radius } from '../theme'

function labelFor(type?: string, url?: string): string {
  if (!type || type === 'http') {
    if (url?.startsWith('https://')) return 'HTTPS'
    if (url?.startsWith('http://')) return 'HTTP'
    return 'HTTPS'
  }
  if (type === 'performance' || type === 'perf') return 'PERF'
  const labels: Record<string, string> = {
    port: 'PORT',
    ssl: 'SSL',
    dns: 'DNS',
    heartbeat: 'HEARTBEAT',
  }
  return labels[type] || type.toUpperCase()
}

export default function CheckTypePill({ type, url }: { type?: string; url?: string }) {
  const kind = monitorKindFor(type)
  const color = kindColor(kind)
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        background: `color-mix(in srgb, ${color} 16%, transparent)`,
        color,
        padding: '3px 8px 3px 6px',
        borderRadius: radius.sm,
        fontSize: 11,
        fontWeight: 700,
        letterSpacing: '0.04em',
        border: `1px solid color-mix(in srgb, ${color} 35%, transparent)`,
        whiteSpace: 'nowrap',
      }}
    >
      <MonitorKindIcon kind={kind} size={12} color={color} />
      {labelFor(type, url)}
    </span>
  )
}
