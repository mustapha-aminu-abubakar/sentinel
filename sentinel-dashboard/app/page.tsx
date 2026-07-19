'use client'

import { useMemo } from 'react'
import useSWR from 'swr'
import { Activity, AlertCircle, CheckCircle, XCircle, Percent, Clock } from 'lucide-react'
import { getUsage, getLatency, sentinelKeys } from '@/lib/api/sentinel'
import { aggregateUsage, weightedAvgLatency } from '@/lib/overview/aggregate'
import { KpiCard } from '@/components/overview/KpiCard'
import { Badge } from '@/components/ui/badge'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

function defaultWindow() {
  const to = new Date()
  const from = new Date(to.getTime() - 24 * 60 * 60 * 1000)
  return { from: from.toISOString(), to: to.toISOString() }
}

function LoadingCard() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <Skeleton className="h-4 w-24" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-8 w-20" />
      </CardContent>
    </Card>
  )
}

export default function OverviewPage() {
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
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Failed to load metrics</AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : 'Unknown error'}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const agg = usage ? aggregateUsage(usage) : null
  const avgLat = latency ? weightedAvgLatency(latency) : null

  return (
    <div className="space-y-4">
      <div className="flex items-baseline gap-3">
        <h1 className="text-3xl font-bold tracking-tight">Overview</h1>
        <Badge variant="outline" className="shrink-0">
          Last 24 hours
        </Badge>
      </div>
      <p className="text-muted-foreground">Key performance metrics.</p>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {loading
          ? Array.from({ length: 5 }).map((_, i) => <LoadingCard key={i} />)
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
