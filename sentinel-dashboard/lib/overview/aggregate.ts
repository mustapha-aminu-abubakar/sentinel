import type { UsagePoint, LatencyPoint } from '@/lib/api/types'

export function aggregateUsage(points: UsagePoint[]) {
  const total = points.reduce((s, p) => s + p.allowed + p.rejected, 0)
  const allowed = points.reduce((s, p) => s + p.allowed, 0)
  const rejected = points.reduce((s, p) => s + p.rejected, 0)
  const successPct = total > 0 ? (allowed / total) * 100 : 0
  return { total, allowed, rejected, successPct }
}

export function weightedAvgLatency(points: LatencyPoint[]): number {
  if (points.length === 0) return 0
  const total = points.reduce((s, p) => s + p.avg_latency_ms, 0)
  return total / points.length
}
