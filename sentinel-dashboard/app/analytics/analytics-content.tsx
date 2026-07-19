'use client'

import { useCallback, useMemo } from 'react'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import useSWR from 'swr'
import { AlertCircle } from 'lucide-react'
import { getUsage, getLatency, sentinelKeys } from '@/lib/api/sentinel'
import type { UsageFilters } from '@/lib/api/types'
import { ApiError } from '@/lib/api/types'
import { rangeToWindow, type Range } from '@/lib/analytics/range'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { FiltersBar, type FiltersState } from '@/components/analytics/FiltersBar'
import { RequestVolumeChart } from '@/components/analytics/RequestVolumeChart'
import { LatencyChart } from '@/components/analytics/LatencyChart'

const RANGE_LABELS: Record<Range, string> = {
  '1d': 'Past 24 hours',
  '7d': 'Past 7 days',
  '30d': 'Past 30 days',
}

function readFilters(sp: URLSearchParams): FiltersState {
  const range = sp.get('range')
  return {
    clientId: sp.get('clientId') ?? '',
    api: sp.get('api') ?? '',
    range: range === '1d' || range === '7d' || range === '30d' ? range : '30d',
    status: sp.get('status') ?? 'all',
  }
}

export default function AnalyticsContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const filters = useMemo(() => readFilters(searchParams), [searchParams])

  const createQueryString = useCallback(
    (patch: Partial<FiltersState>) => {
      const params = new URLSearchParams(searchParams.toString())
      for (const [key, value] of Object.entries(patch)) {
        if (value === '' || value === 'all') {
          params.delete(key)
        } else {
          params.set(key, value)
        }
      }
      return params.toString()
    },
    [searchParams]
  )

  const onFilterChange = useCallback(
    (patch: Partial<FiltersState>) => {
      router.push(`${pathname}?${createQueryString(patch)}`, { scroll: false })
    },
    [router, pathname, createQueryString]
  )

  const timeWindow = useMemo(
    () => rangeToWindow(filters.range),
    [filters.range]
  )

  const usageFilters: UsageFilters = useMemo(
    () => ({
      ...(filters.clientId && { clientId: filters.clientId }),
      ...(filters.api && { api: filters.api }),
      ...(timeWindow.from && { from: timeWindow.from }),
      ...(timeWindow.to && { to: timeWindow.to }),
      ...(filters.status !== 'all' && {
        status: filters.status as 'allowed' | 'rejected',
      }),
    }),
    [filters, timeWindow]
  )

  const latencyFilters = useMemo(() => {
    const { status, ...rest } = usageFilters
    void status
    return rest
  }, [usageFilters])

  const { data: usageData, error: usageError, isLoading: usageLoading } = useSWR(
    sentinelKeys.usage(usageFilters),
    () => getUsage(usageFilters),
    { keepPreviousData: true }
  )

  const { data: latencyData, error: latencyError, isLoading: latencyLoading } = useSWR(
    sentinelKeys.latency(latencyFilters),
    () => getLatency(latencyFilters),
    { keepPreviousData: true }
  )

  const statusFilter = usageFilters.status as 'allowed' | 'rejected' | undefined

  return (
    <div className="space-y-4">
      <div className="flex items-baseline gap-3">
        <h1 className="text-3xl font-bold tracking-tight">Usage Analytics</h1>
        <Badge variant="outline" className="shrink-0">
          {RANGE_LABELS[filters.range]}
        </Badge>
      </div>
      <p className="text-muted-foreground">
        Request volume and latency trends over time.
      </p>
      <FiltersBar filters={filters} onFilterChange={onFilterChange} />
      {usageError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Failed to load usage data</AlertTitle>
          <AlertDescription>
            {usageError instanceof ApiError
              ? `${usageError.status}: ${usageError.message}`
              : 'Network error — is the API server running?'}
          </AlertDescription>
        </Alert>
      )}
      {latencyError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Failed to load latency data</AlertTitle>
          <AlertDescription>
            {latencyError instanceof ApiError
              ? `${latencyError.status}: ${latencyError.message}`
              : 'Network error — is the API server running?'}
          </AlertDescription>
        </Alert>
      )}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Request Volume</CardTitle>
          </CardHeader>
          <CardContent>
            <RequestVolumeChart
              data={usageData ?? []}
              loading={usageLoading}
              statusFilter={statusFilter}
              range={filters.range}
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Latency</CardTitle>
          </CardHeader>
          <CardContent>
            <LatencyChart
              data={latencyData ?? []}
              loading={latencyLoading}
              range={filters.range}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
