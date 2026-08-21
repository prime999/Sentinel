/** Parse a number input so clearing the field stays empty instead of becoming 0. */
export function parseNumberInput(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (trimmed === '') return undefined
  const n = Number(trimmed)
  return Number.isFinite(n) ? n : undefined
}

export function numberFieldValue(n: number | undefined | null): number | '' {
  return n == null || Number.isNaN(n) ? '' : n
}
