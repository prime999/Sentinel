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

  return (
    <div style={{
      background: colors.card,
      border: `1px solid ${colors.border}`,
      borderRadius: colors.radius,
      padding: 24,
    }}>
      <div style={{
        fontSize: 12,
        fontWeight: 500,
        color: colors.textMuted,
        marginBottom: 10,
      }}>
        {label}
      </div>
      <div style={{
        fontSize: 28,
        fontWeight: 700,
        color: accentColor,
        lineHeight: 1.1,
        letterSpacing: '-0.02em',
      }}>
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: 12, color: colors.textMuted, marginTop: 8 }}>
          {sub}
        </div>
      )}
    </div>
  )
}
