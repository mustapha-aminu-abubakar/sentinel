import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  qs,
  listClients,
  createClient,
  updateClient,
  listRules,
  createRule,
  updateRule,
  getUsage,
  getLatency,
  sentinelKeys,
} from './sentinel'
import { ApiError } from './types'

const BASE = 'http://test.example.com'

beforeEach(() => {
  vi.stubEnv('NEXT_PUBLIC_SENTINEL_API_URL', BASE)
})

function mockFetch(status: number, body: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 400 ? 'Bad Request' : 'Internal Server Error',
    json: () => Promise.resolve(body),
    text: () =>
      Promise.resolve(
        typeof body === 'string' ? body : JSON.stringify(body)
      ),
  } as Response)
}

describe('qs', () => {
  it('skips undefined values', () => {
    expect(qs({ a: 1, b: undefined, c: 'x' })).toBe('?a=1&c=x')
  })

  it('skips null values', () => {
    expect(qs({ a: null, b: 'y' })).toBe('?b=y')
  })

  it('returns empty string for empty input', () => {
    expect(qs({})).toBe('')
  })

  it('encodes special characters', () => {
    expect(qs({ name: 'hello world', q: 'a&b' })).toBe('?name=hello+world&q=a%26b')
  })
})

describe('listClients', () => {
  it('calls GET /clients and returns clients', async () => {
    const fetch = mockFetch(200, [{ id: '1', name: 'Test' }])
    const result = await listClients()
    expect(fetch).toHaveBeenCalledWith(`${BASE}/clients`, expect.any(Object))
    expect(result).toEqual([{ id: '1', name: 'Test' }])
  })
})

describe('createClient', () => {
  it('calls POST /clients with name', async () => {
    const fetch = mockFetch(201, { id: '1', name: 'New', status: 'active', createdAt: '', updatedAt: '' })
    const result = await createClient({ name: 'New' })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/clients`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'New' }),
    })
    expect(result.name).toBe('New')
  })
})

describe('updateClient', () => {
  it('calls PATCH /clients/:id', async () => {
    const fetch = mockFetch(200, { id: '1', name: 'Updated', status: 'inactive', createdAt: '', updatedAt: '' })
    const result = await updateClient('1', { name: 'Updated', status: 'inactive' })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/clients/1`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Updated', status: 'inactive' }),
    })
    expect(result.name).toBe('Updated')
  })
})

describe('listRules', () => {
  it('calls GET /rules', async () => {
    const fetch = mockFetch(200, [])
    await listRules()
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules`, expect.any(Object))
  })

  it('appends clientId query param', async () => {
    const fetch = mockFetch(200, [])
    await listRules({ clientId: 'c1' })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules?clientId=c1`, expect.any(Object))
  })
})

describe('createRule', () => {
  it('calls POST /rules with payload', async () => {
    const payload = { clientId: 'c1', api: 'stripe', requestsAllowed: 100, windowSeconds: 60 }
    const fetch = mockFetch(201, { id: 'r1', ...payload, createdAt: '', updatedAt: '' })
    const result = await createRule(payload)
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    expect(result.id).toBe('r1')
  })
})

describe('updateRule', () => {
  it('calls PATCH /rules/:id', async () => {
    const fetch = mockFetch(200, { id: 'r1', requestsAllowed: 500 })
    await updateRule('r1', { requestsAllowed: 500 })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules/r1`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ requestsAllowed: 500 }),
    })
  })
})

describe('getUsage', () => {
  it('calls GET /analytics/usage with query params', async () => {
    const fetch = mockFetch(200, [])
    await getUsage({ clientId: 'c1', from: '2024-01-01', to: '2024-01-07' })
    expect(fetch).toHaveBeenCalledWith(
      `${BASE}/analytics/usage?clientId=c1&from=2024-01-01&to=2024-01-07`,
      expect.any(Object)
    )
  })
})

describe('getLatency', () => {
  it('calls GET /analytics/latency with query params', async () => {
    const fetch = mockFetch(200, [])
    await getLatency({ clientId: 'c1', from: '2024-01-01', to: '2024-01-07' })
    expect(fetch).toHaveBeenCalledWith(
      `${BASE}/analytics/latency?clientId=c1&from=2024-01-01&to=2024-01-07`,
      expect.any(Object)
    )
  })
})

describe('ApiError', () => {
  it('throws ApiError on non-2xx response', async () => {
    mockFetch(400, 'client error')
    await expect(listClients()).rejects.toThrow(ApiError)
    await expect(listClients()).rejects.toThrow('client error')
  })

  it('throws ApiError with status code on 500', async () => {
    mockFetch(500, 'server error')
    try {
      await listClients()
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError)
      expect((e as ApiError).status).toBe(500)
    }
  })
})

describe('sentinelKeys', () => {
  it('returns stable cache keys', () => {
    expect(sentinelKeys.clients()).toBe('/clients')
    expect(sentinelKeys.rules()).toBe('/rules')
    expect(sentinelKeys.rules({ clientId: 'c1' })).toBe('/rules?clientId=c1')
    expect(sentinelKeys.usage({ clientId: 'c1' })).toBe('/analytics/usage?clientId=c1')
    expect(sentinelKeys.latency({ from: '2024-01-01' })).toBe('/analytics/latency?from=2024-01-01')
  })
})
