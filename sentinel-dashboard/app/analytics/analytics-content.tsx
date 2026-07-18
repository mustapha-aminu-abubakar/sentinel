'use client'

import { useCallback, useMemo } from 'react'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import useSWR from 'swr'
import { getUsage, getLatency, sentinelKeys } from '@/lib/api/sentinel'
import type { UsageFilters } from '@/lib/api/types'
import { rangeToWindow, type Range } from '@/lib/analytics/range'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { FiltersBar, type FiltersState } from '@/components/analytics/FiltersBar'
import { RequestVolumeChart } from '@/components/analytics/RequestVolumeChart'
import { LatencyChart } from '@/components/analytics/LatencyChart'

function readFilters(sp: URLSearchParams): FiltersState {
  const range = sp.get('range')
  return {
    clientId: sp.get('clientId') ?? '',
    api: sp.get('api') ?? '',
    range:
      range === '10d' || range === '15d' || range === '30d' || range === 'custom'
        ? range
        : '30d',
    from: sp.get('from') ?? '',
    to: sp.get('to') ?? '',
    status: sp.get('status') ?? 'all',
  }
}

/** Safely convert a date string to ISO — returns '' on invalid input. */
function safeToISO(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return isNaN(d.getTime()) ? '' : d.toISOString()
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

  const timeWindow = useMemo(() => {
    if (filters.range === 'custom') {
      return {
        from: safeToISO(filters.from),
        to: safeToISO(filters.to),
      }
    }
    return rangeToWindow(filters.range as Range)
  }, [filters.range, filters.from, filters.to])

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
    const { status: _, ...rest } = usageFilters
    return rest
  }, [usageFilters])

  // Use isLoading (not isValidating) so keepPreviousData can show stale data
  // during background revalidation without triggering a skeleton flash.
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
      <h1 className="text-3xl font-bold tracking-tight">Usage Analytics</h1>
      <p className="text-muted-foreground">
        Request volume and latency trends over time.
      </p>
      <FiltersBar filters={filters} onFilterChange={onFilterChange} />
      {usageError && (
        <p className="text-sm text-destructive">
          Failed to load usage data. Please try again.
        </p>
      )}
      {latencyError && (
        <p className="text-sm text-destructive">
          Failed to load latency data. Please try again.
        </p>
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
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
