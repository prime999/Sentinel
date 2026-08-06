import { colors } from '../theme'

const statusStyles: Record<string, { bg: string; text: string; dot: string }> = {
  up: { bg: colors.greenDim, text: colors.green, dot: colors.green },
  down: { bg: colors.redDim, text: colors.red, dot: colors.red },
  degraded: { bg: colors.yellowDim, text: colors.yellow, dot: colors.yellow },
  unknown: { bg: 'rgba(139,148,158,0.15)', text: colors.textMuted, dot: colors.textMuted },
}

export default function StatusBadge({ status }: { status: string }) {
  const c = statusStyles[status] || statusStyles.unknown
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6,
      background: c.bg,
      color: c.text,
      padding: '5px 12px',
      borderRadius: 20,
      fontSize: 12,
      fontWeight: 600,
      textTransform: 'uppercase',
      letterSpacing: '0.04em',
    }}>
      <span style={{ width: 7, height: 7, borderRadius: '50%', background: c.dot }} />
      {status}
    </span>
  )
}
