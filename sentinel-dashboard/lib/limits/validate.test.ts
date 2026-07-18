import { describe, it, expect } from 'vitest'
import { validateRuleInput, validateClientInput } from './validate'

describe('validateRuleInput', () => {
  it('returns errors for empty api', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: '',
      requestsAllowed: 100,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.api).toBe('API identifier is required')
    }
  })

  it('returns errors for requestsAllowed <= 0', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 0,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.requestsAllowed).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns errors for negative requestsAllowed', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: -1,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.requestsAllowed).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns errors for float requestsAllowed', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 1.5,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.requestsAllowed).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns errors for requestsAllowed exceeding maximum', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 10_000_001,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.requestsAllowed).toMatch(/10,000,000/)
    }
  })

  it('returns errors for windowSeconds <= 0', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: -1,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.windowSeconds).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns errors for windowSeconds = 0', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: 0,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.windowSeconds).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns errors for windowSeconds exceeding maximum', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: 86_401,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.windowSeconds).toMatch(/86,400/)
    }
  })

  it('returns errors for api exceeding max length', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'a'.repeat(256),
      requestsAllowed: 100,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.api).toMatch(/255/)
    }
  })

  it('returns errors for empty clientId', () => {
    const result = validateRuleInput({
      clientId: '',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.clientId).toBe('Client is required')
    }
  })

  it('returns errors for all invalid fields simultaneously', () => {
    const result = validateRuleInput({
      clientId: '',
      api: '',
      requestsAllowed: 0,
      windowSeconds: -1,
    })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.clientId).toBe('Client is required')
      expect(result.errors.api).toBe('API identifier is required')
      expect(result.errors.requestsAllowed).toBe('Must be a whole number greater than 0')
      expect(result.errors.windowSeconds).toBe('Must be a whole number greater than 0')
    }
  })

  it('returns ok for valid input', () => {
    const result = validateRuleInput({
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: 60,
    })
    expect(result.ok).toBe(true)
  })
})

describe('validateClientInput', () => {
  it('returns error for empty name', () => {
    const result = validateClientInput({ name: '' })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.name).toBe('Client name is required')
    }
  })

  it('returns error for whitespace-only name', () => {
    const result = validateClientInput({ name: '   ' })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.name).toBe('Client name is required')
    }
  })

  it('returns error for name exceeding max length', () => {
    const result = validateClientInput({ name: 'x'.repeat(256) })
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.name).toMatch(/255/)
    }
  })

  it('returns ok for valid name', () => {
    const result = validateClientInput({ name: 'Stripe Integration' })
    expect(result.ok).toBe(true)
  })
})
