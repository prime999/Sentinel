import { colors } from '../theme'

export default function MetricCard({
  label,
  value,
  sub,
  accent = 'default',
}: {
  label: string
  value: string
  sub?: string
  accent?: 'default' | 'green' | 'blue' | 'yellow' | 'red'
}) {
  const accentColor = {
    default: colors.text,
    green: colors.green,
    blue: colors.blue,
    yellow: colors.yellow,
    red: colors.red,
  }[accent]

  const accentBg = {
    default: colors.card,
    green: colors.greenDim,
    blue: colors.blueDim,
    yellow: colors.yellowDim,
    red: colors.redDim,
  }[accent]

  return (
    <div style={{
      background: accentBg,
      border: `1px solid ${colors.border}`,
      borderRadius: 12,
      padding: '18px 20px',
    }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: colors.textMuted, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 8 }}>
        {label}
      </div>
      <div style={{ fontSize: 28, fontWeight: 700, color: accentColor, lineHeight: 1.1 }}>
        {value}
      </div>
      {sub && <div style={{ fontSize: 12, color: colors.textDim, marginTop: 6 }}>{sub}</div>}
    </div>
  )
}
