import type { Range } from './range'

export function formatBucket(bucket: string, range?: Range): string {
  const d = new Date(bucket)
  if (isNaN(d.getTime())) return bucket
  if (range === '1d') {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function tickCount(range?: Range): number {
  switch (range) {
    case '1d':
      return 8
    case '7d':
      return 7
    case '30d':
      return 10
    default:
      return 5
  }
}
