export type Range = '10d' | '15d' | '30d'

export function rangeToWindow(
  range: Range,
  now: Date = new Date()
): { from: string; to: string } {
  const days = { '10d': 10, '15d': 15, '30d': 30 }[range]
  const from = new Date(now)
  // Use UTC arithmetic to avoid DST-transition bugs (CodeRabbit finding)
  from.setUTCDate(from.getUTCDate() - days)
  return {
    from: from.toISOString(),
    to: now.toISOString(),
  }
}
