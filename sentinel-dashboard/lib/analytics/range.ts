export type Range = '1d' | '7d' | '30d'

export function rangeToWindow(
  range: Range,
  now: Date = new Date()
): { from: string; to: string } {
  const days = { '1d': 1, '7d': 7, '30d': 30 }[range]
  const from = new Date(now)
  from.setUTCDate(from.getUTCDate() - days)
  return {
    from: from.toISOString(),
    to: now.toISOString(),
  }
}
