'use client'

import { useMemo } from 'react'
import useSWR from 'swr'
import { Activity, CheckCircle, XCircle, Percent, Clock } from 'lucide-react'
import { getUsage, getLatency, sentinelKeys } from '@/lib/api/sentinel'
import { aggregateUsage, weightedAvgLatency } from '@/lib/overview/aggregate'
import { KpiCard } from '@/components/overview/KpiCard'
import { Skeleton } from '@/components/ui/skeleton'

// Computed once per mount — stable reference for SWR key
function defaultWindow() {
  const to = new Date()
  const from = new Date(to.getTime() - 24 * 60 * 60 * 1000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export default function OverviewPage() {
  // useMemo prevents a new object on every render which would cause an
  // infinite SWR re-fetch loop (new key object === new fetch).
  const filters = useMemo(() => defaultWindow(), [])

  const { data: usage, error: usageErr, isLoading: usageLoad } = useSWR(
    sentinelKeys.usage(filters),
    () => getUsage(filters),
  )
  const { data: latency, error: latencyErr, isLoading: latencyLoad } = useSWR(
    sentinelKeys.latency(filters),
    () => getLatency(filters),
  )

  const loading = usageLoad || latencyLoad
  const error = usageErr || latencyErr

  if (error) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold tracking-tight">Overview</h1>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-destructive">
          <p className="font-medium">Failed to load metrics</p>
          <p className="text-sm">{error instanceof Error ? error.message : 'Unknown error'}</p>
        </div>
      </div>
    )
  }

  // Derive aggregates — may be null while data is still loading
  const agg = usage ? aggregateUsage(usage) : null
  const avgLat = latency ? weightedAvgLatency(latency) : null

  return (
    <div className="space-y-4">
      <h1 className="text-3xl font-bold tracking-tight">Overview</h1>
      <p className="text-muted-foreground">Key performance metrics for the last 24 hours.</p>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {loading
          ? Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-[120px] rounded-xl" />
            ))
          : [
              <KpiCard key="total" label="Total Requests" value={(agg?.total ?? 0).toLocaleString()} icon={<Activity className="h-4 w-4 text-muted-foreground" />} />,
              <KpiCard key="allowed" label="Allowed" value={(agg?.allowed ?? 0).toLocaleString()} icon={<CheckCircle className="h-4 w-4 text-emerald-500" />} />,
              <KpiCard key="rejected" label="Rejected" value={(agg?.rejected ?? 0).toLocaleString()} icon={<XCircle className="h-4 w-4 text-red-500" />} />,
              <KpiCard key="success" label="Success %" value={(agg?.successPct ?? 0).toFixed(1)} unit="%" icon={<Percent className="h-4 w-4 text-muted-foreground" />} />,
              <KpiCard key="latency" label="Avg Latency" value={avgLat != null ? avgLat.toFixed(0) : '—'} unit={avgLat != null ? 'ms' : undefined} icon={<Clock className="h-4 w-4 text-muted-foreground" />} />,
            ]}
      </div>
    </div>
  )
}
