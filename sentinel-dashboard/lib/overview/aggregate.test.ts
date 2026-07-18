import { describe, it, expect } from 'vitest'
import { aggregateUsage, weightedAvgLatency } from './aggregate'
import type { UsagePoint, LatencyPoint } from '@/lib/api/types'

describe('aggregateUsage', () => {
  it('returns zeros for empty array', () => {
    const result = aggregateUsage([])
    expect(result).toEqual({ total: 0, allowed: 0, rejected: 0, successPct: 0 })
  })

  it('sums allowed and rejected correctly', () => {
    const points: UsagePoint[] = [
      { bucket: '2026-01-01T00:00:00Z', allowed: 10, rejected: 2, avgLatencyMs: 5 },
      { bucket: '2026-01-01T01:00:00Z', allowed: 5, rejected: 1, avgLatencyMs: 8 },
    ]
    const result = aggregateUsage(points)
    expect(result.total).toBe(18)
    expect(result.allowed).toBe(15)
    expect(result.rejected).toBe(3)
    expect(result.successPct).toBeCloseTo(83.33, 1)
  })

  it('returns zero successPct when total is zero', () => {
    const points: UsagePoint[] = [
      { bucket: '2026-01-01T00:00:00Z', allowed: 0, rejected: 0, avgLatencyMs: 0 },
    ]
    const result = aggregateUsage(points)
    expect(result.total).toBe(0)
    expect(result.successPct).toBe(0)
  })

  it('handles all-rejected', () => {
    const points: UsagePoint[] = [
      { bucket: '2026-01-01T00:00:00Z', allowed: 0, rejected: 10, avgLatencyMs: 0 },
    ]
    const result = aggregateUsage(points)
    expect(result.total).toBe(10)
    expect(result.allowed).toBe(0)
    expect(result.rejected).toBe(10)
    expect(result.successPct).toBe(0)
  })
})

describe('weightedAvgLatency', () => {
  it('returns zero for empty array', () => {
    expect(weightedAvgLatency([])).toBe(0)
  })

  it('returns the value for a single point', () => {
    const points: LatencyPoint[] = [
      { bucket: '2026-01-01T00:00:00Z', avgLatencyMs: 42, p95LatencyMs: 50 },
    ]
    expect(weightedAvgLatency(points)).toBe(42)
  })

  it('computes average of multiple points', () => {
    const points: LatencyPoint[] = [
      { bucket: '2026-01-01T00:00:00Z', avgLatencyMs: 5, p95LatencyMs: 10 },
      { bucket: '2026-01-01T01:00:00Z', avgLatencyMs: 15, p95LatencyMs: 20 },
    ]
    expect(weightedAvgLatency(points)).toBe(10)
  })
})
