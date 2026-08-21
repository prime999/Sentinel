import { colors, fonts, radius } from './theme'

export const chartTick = {
  fill: colors.textMuted,
  fontSize: 10,
  fontFamily: fonts.mono,
}

export const chartTooltipStyle = {
  background: colors.card,
  border: `1px solid ${colors.border}`,
  borderRadius: radius.md,
  color: colors.text,
  fontSize: 12,
  fontFamily: fonts.sans,
}

export const chartTooltipLabel = {
  color: colors.textMuted,
}

export const chartGridStroke = colors.border
