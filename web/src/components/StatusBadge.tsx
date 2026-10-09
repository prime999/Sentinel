import {
  AlertTriangle,
  CircleCheck,
  CircleHelp,
  CirclePause,
  CircleX,
  Clock,
  type LucideIcon,
} from 'lucide-react'
import { colors, radius } from '../theme'

const labels: Record<string, string> = {
  up: 'Healthy',
  down: 'Down',
  degraded: 'Warning',
  unknown: 'Unknown',
  pending: 'Waiting',
  online: 'Online',
  offline: 'Offline',
  critical: 'Critical',
  paused: 'Paused',
}

const statusMeta: Record<string, { bg: string; text: string; Icon: LucideIcon }> = {
  up: { bg: colors.greenDim, text: colors.green, Icon: CircleCheck },
  down: { bg: colors.redDim, text: colors.red, Icon: CircleX },
  degraded: { bg: colors.yellowDim, text: colors.yellow, Icon: AlertTriangle },
  critical: { bg: colors.redDim, text: colors.red, Icon: CircleX },
  paused: { bg: 'rgba(156,163,175,0.12)', text: colors.textMuted, Icon: CirclePause },
  pending: { bg: colors.blueDim, text: colors.blue, Icon: Clock },
  online: { bg: colors.greenDim, text: colors.green, Icon: CircleCheck },
  offline: { bg: colors.redDim, text: colors.red, Icon: CircleX },
  unknown: { bg: 'rgba(156,163,175,0.12)', text: colors.textMuted, Icon: CircleHelp },
}

export function isPaused(target: { enabled?: boolean }): boolean {
  return target.enabled === false
}

/** Map stored monitor status to badge status (SSL down = critical expiry). */
export function badgeStatusFor(monitorType: string | undefined, status: string, enabled?: boolean): string {
  if (enabled === false) return 'paused'
  if ((monitorType || 'http') === 'ssl' && status === 'down') return 'critical'
  return status
}

export default function StatusBadge({ status }: { status: string }) {
  const c = statusMeta[status] || statusMeta.unknown
  const Icon = c.Icon
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6,
      background: c.bg,
      color: c.text,
      padding: '4px 8px',
      borderRadius: radius.sm,
      fontSize: 13,
      fontWeight: 600,
      letterSpacing: '0.02em',
    }}>
      <Icon size={14} color={c.text} strokeWidth={2} aria-hidden />
      {labels[status] || status}
    </span>
  )
}
