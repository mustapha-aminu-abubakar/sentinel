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
  it('unwraps envelope and transforms snake_case keys', async () => {
    const fetch = mockFetch(200, {
      clients: [
        {
          id: '1',
          name: 'Test',
          status: 'active',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
        },
      ],
    })
    const result = await listClients()
    expect(fetch).toHaveBeenCalledWith(`${BASE}/clients`, expect.any(Object))
    expect(result).toEqual([
      {
        id: '1',
        name: 'Test',
        status: 'active',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-02T00:00:00Z',
      },
    ])
  })
})

describe('createClient', () => {
  it('calls POST /clients with name', async () => {
    const fetch = mockFetch(201, { id: '1', name: 'New', status: 'active', created_at: '', updated_at: '' })
    const result = await createClient({ name: 'New' })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/clients`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'New' }),
    })
    expect(result.name).toBe('New')
    expect(result.createdAt).toBe('')
  })
})

describe('updateClient', () => {
  it('calls PATCH /clients/:id', async () => {
    const fetch = mockFetch(200, { id: '1', name: 'Updated', status: 'inactive', created_at: '', updated_at: '' })
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
  it('unwraps envelope and transforms snake_case keys', async () => {
    const mockRule = {
      id: 'r1',
      client_id: 'c1',
      api: 'stripe',
      requests_allowed: 100,
      window_seconds: 60,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-02T00:00:00Z',
    }
    const fetch = mockFetch(200, { rules: [mockRule] })
    const result = await listRules()
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules`, expect.any(Object))
    expect(result).toEqual([
      {
        id: 'r1',
        clientId: 'c1',
        api: 'stripe',
        requestsAllowed: 100,
        windowSeconds: 60,
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-02T00:00:00Z',
      },
    ])
  })

  it('sends client_id query param', async () => {
    const fetch = mockFetch(200, { rules: [] })
    await listRules({ clientId: 'c1' })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules?client_id=c1`, expect.any(Object))
  })
})

describe('createRule', () => {
  it('sends snake_case body and transforms response', async () => {
    const payload = { clientId: 'c1', api: 'stripe', requestsAllowed: 100, windowSeconds: 60 }
    const fetch = mockFetch(201, {
      id: 'r1',
      client_id: 'c1',
      api: 'stripe',
      requests_allowed: 100,
      window_seconds: 60,
      created_at: '',
      updated_at: '',
    })
    const result = await createRule(payload)
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: 'c1',
        api: 'stripe',
        requests_allowed: 100,
        window_seconds: 60,
      }),
    })
    expect(result).toEqual({
      id: 'r1',
      clientId: 'c1',
      api: 'stripe',
      requestsAllowed: 100,
      windowSeconds: 60,
      createdAt: '',
      updatedAt: '',
    })
  })
})

describe('updateRule', () => {
  it('sends snake_case body and transforms response', async () => {
    const fetch = mockFetch(200, {
      id: 'r1',
      requests_allowed: 500,
      window_seconds: 30,
    })
    const result = await updateRule('r1', { requestsAllowed: 500, windowSeconds: 30 })
    expect(fetch).toHaveBeenCalledWith(`${BASE}/rules/r1`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ requests_allowed: 500, window_seconds: 30 }),
    })
    expect(result).toMatchObject({
      id: 'r1',
      requestsAllowed: 500,
      windowSeconds: 30,
    })
  })
})

describe('getUsage', () => {
  it('sends snake_case query params', async () => {
    const fetch = mockFetch(200, [])
    await getUsage({ clientId: 'c1', from: '2024-01-01', to: '2024-01-07' })
    expect(fetch).toHaveBeenCalledWith(
      `${BASE}/analytics/usage?client_id=c1&from=2024-01-01&to=2024-01-07`,
      expect.any(Object)
    )
  })
})

describe('getLatency', () => {
  it('sends snake_case query params', async () => {
    const fetch = mockFetch(200, [])
    await getLatency({ clientId: 'c1', from: '2024-01-01', to: '2024-01-07' })
    expect(fetch).toHaveBeenCalledWith(
      `${BASE}/analytics/latency?client_id=c1&from=2024-01-01&to=2024-01-07`,
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
    expect(sentinelKeys.rules({ clientId: 'c1' })).toBe('/rules?client_id=c1')
    expect(sentinelKeys.usage({ clientId: 'c1' })).toBe('/analytics/usage?client_id=c1')
    expect(sentinelKeys.latency({ from: '2024-01-01' })).toBe('/analytics/latency?from=2024-01-01')
  })
})
