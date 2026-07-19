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
  allowed: number
  rejected: number
}

export interface LatencyPoint {
  bucket: string
  avgLatencyMs: number
  p95LatencyMs: number
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
