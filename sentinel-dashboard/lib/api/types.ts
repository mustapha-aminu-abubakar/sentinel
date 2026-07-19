export interface Client {
  id: string
  name: string
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
}

export interface RateRule {
  id: string
  clientId: string
  api: string
  requestsAllowed: number
  windowSeconds: number
  createdAt: string
  updatedAt: string
}

export interface UsagePoint {
  bucket: string
  clientId?: string
  api?: string
  allowed: number
  rejected: number
  avgLatencyMs: number
}

export interface LatencyPoint {
  bucket: string
  avg_latency_ms: number
  p95_latency_ms: number
}

export interface UsageFilters {
  clientId?: string
  api?: string
  from?: string
  to?: string
  status?: 'allowed' | 'rejected'
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}
