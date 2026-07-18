import type { Client, RateRule, UsagePoint, LatencyPoint, UsageFilters } from './types'
import { ApiError } from './types'

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
  return fetcher<Client[]>('/clients')
}

export function createClient(input: { name: string }): Promise<Client> {
  return fetcher<Client>('/clients', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateClient(
  id: string,
  patch: Partial<Pick<Client, 'name' | 'status'>>
): Promise<Client> {
  return fetcher<Client>(`/clients/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export function listRules(params?: { clientId?: string }): Promise<RateRule[]> {
  const query = params?.clientId ? qs({ clientId: params.clientId }) : ''
  return fetcher<RateRule[]>(`/rules${query}`)
}

export function createRule(input: {
  clientId: string
  api: string
  requestsAllowed: number
  windowSeconds: number
}): Promise<RateRule> {
  return fetcher<RateRule>('/rules', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateRule(
  id: string,
  patch: Partial<Pick<RateRule, 'requestsAllowed' | 'windowSeconds' | 'api'>>
): Promise<RateRule> {
  return fetcher<RateRule>(`/rules/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export function getUsage(filters: UsageFilters): Promise<UsagePoint[]> {
  return fetcher<UsagePoint[]>(`/analytics/usage${qs(filters as Record<string, unknown>)}`)
}

export function getLatency(
  filters: Omit<UsageFilters, 'status'>
): Promise<LatencyPoint[]> {
  return fetcher<LatencyPoint[]>(
    `/analytics/latency${qs(filters as Record<string, unknown>)}`
  )
}

export const sentinelKeys = {
  clients: () => '/clients' as const,
  rules: (params?: { clientId?: string }) =>
    params?.clientId ? `/rules?clientId=${params.clientId}` : '/rules',
  usage: (filters: UsageFilters) => `/analytics/usage${qs(filters as Record<string, unknown>)}`,
  latency: (filters: Omit<UsageFilters, 'status'>) =>
    `/analytics/latency${qs(filters as Record<string, unknown>)}`,
}
