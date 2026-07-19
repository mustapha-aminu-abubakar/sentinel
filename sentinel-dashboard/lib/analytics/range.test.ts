import { describe, it, expect } from 'vitest'
import { rangeToWindow } from './range'

describe('rangeToWindow', () => {
  it('returns a 1-day window from a fixed date', () => {
    const now = new Date('2025-01-20T00:00:00.000Z')
    const result = rangeToWindow('1d', now)
    expect(result.from).toBe('2025-01-19T00:00:00.000Z')
    expect(result.to).toBe('2025-01-20T00:00:00.000Z')
  })

  it('returns a 7-day window from a fixed date', () => {
    const now = new Date('2025-01-20T00:00:00.000Z')
    const result = rangeToWindow('7d', now)
    expect(result.from).toBe('2025-01-13T00:00:00.000Z')
    expect(result.to).toBe('2025-01-20T00:00:00.000Z')
  })

  it('returns a 30-day window from a fixed date', () => {
    const now = new Date('2025-01-31T00:00:00.000Z')
    const result = rangeToWindow('30d', now)
    expect(result.from).toBe('2025-01-01T00:00:00.000Z')
    expect(result.to).toBe('2025-01-31T00:00:00.000Z')
  })

  it('handles month boundaries correctly', () => {
    const now = new Date('2025-03-01T00:00:00.000Z')
    const result = rangeToWindow('1d', now)
    expect(result.from).toBe('2025-02-28T00:00:00.000Z')
    expect(result.to).toBe('2025-03-01T00:00:00.000Z')
  })

  it('defaults to current date when now is omitted', () => {
    const result = rangeToWindow('30d')
    expect(result.from).toBeTruthy()
    expect(result.to).toBeTruthy()
    expect(new Date(result.from) <= new Date(result.to)).toBe(true)
  })
})
