import type { Client, RateRule, UsagePoint, LatencyPoint, UsageFilters } from './types'
import { ApiError } from './types'

function snakeToCamel(str: string): string {
  return str.replace(/_([a-z])/g, (_, c) => c.toUpperCase())
}

function camelToSnake(str: string): string {
  return str.replace(/[A-Z]/g, c => `_${c.toLowerCase()}`)
}

function mapKeys<T>(obj: unknown, keyFn: (s: string) => string): T {
  if (obj === null || obj === undefined) return obj as T
  if (Array.isArray(obj)) return obj.map(item => mapKeys(item, keyFn)) as T
  if (typeof obj === 'object') {
    const result: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
      result[keyFn(k)] = mapKeys(v, keyFn)
    }
    return result as T
  }
  return obj as T
}

export function qs(params: Record<string, unknown>): string {
  const entries = Object.entries(params).filter(
    ([_, v]) => v !== undefined && v !== null
  )
  if (entries.length === 0) return ''
  return '?' + new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString()
}

function baseUrl(): string {
  return process.env.NEXT_PUBLIC_SENTINEL_API_URL ?? ''
}

async function fetcher<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${baseUrl()}${url}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new ApiError(res.status, body || res.statusText)
  }
  return res.json() as Promise<T>
}

export function listClients(): Promise<Client[]> {
  return fetcher<{ clients: unknown[] }>('/clients')
    .then(r => (r.clients ?? []).map(c => mapKeys<Client>(c, snakeToCamel)))
}

export function createClient(input: { name: string }): Promise<Client> {
  return fetcher<unknown>('/clients', {
    method: 'POST',
    body: JSON.stringify(input),
  }).then(r => mapKeys<Client>(r, snakeToCamel))
}

export function updateClient(
  id: string,
  patch: Partial<Pick<Client, 'name' | 'status'>>
): Promise<Client> {
  return fetcher<unknown>(`/clients/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  }).then(r => mapKeys<Client>(r, snakeToCamel))
}

export function listRules(params?: { clientId?: string }): Promise<RateRule[]> {
  const query = params?.clientId
    ? qs(mapKeys(params, camelToSnake) as Record<string, unknown>)
    : ''
  return fetcher<{ rules: unknown[] }>(`/rules${query}`)
    .then(r => (r.rules ?? []).map(rule => mapKeys<RateRule>(rule, snakeToCamel)))
}

export function createRule(input: {
  clientId: string
  api: string
  requestsAllowed: number
  windowSeconds: number
}): Promise<RateRule> {
  return fetcher<unknown>('/rules', {
    method: 'POST',
    body: JSON.stringify(mapKeys(input, camelToSnake)),
  }).then(r => mapKeys<RateRule>(r, snakeToCamel))
}

export function updateRule(
  id: string,
  patch: Partial<Pick<RateRule, 'requestsAllowed' | 'windowSeconds' | 'api'>>
): Promise<RateRule> {
  return fetcher<unknown>(`/rules/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(mapKeys(patch, camelToSnake)),
  }).then(r => mapKeys<RateRule>(r, snakeToCamel))
}

export function getUsage(filters: UsageFilters): Promise<UsagePoint[]> {
  return fetcher<UsagePoint[]>(
    `/analytics/usage${qs(mapKeys(filters, camelToSnake) as Record<string, unknown>)}`
  )
}

export function getLatency(
  filters: Omit<UsageFilters, 'status'>
): Promise<LatencyPoint[]> {
  return fetcher<unknown[]>(
    `/analytics/latency${qs(mapKeys(filters, camelToSnake) as Record<string, unknown>)}`
  ).then(r => r.map(p => mapKeys<LatencyPoint>(p, snakeToCamel)))
}

export const sentinelKeys = {
  clients: () => '/clients' as const,
  rules: (params?: { clientId?: string }) =>
    params?.clientId ? `/rules?client_id=${params.clientId}` : '/rules',
  usage: (filters: UsageFilters) =>
    `/analytics/usage${qs(mapKeys(filters, camelToSnake) as Record<string, unknown>)}`,
  latency: (filters: Omit<UsageFilters, 'status'>) =>
    `/analytics/latency${qs(mapKeys(filters, camelToSnake) as Record<string, unknown>)}`,
}
