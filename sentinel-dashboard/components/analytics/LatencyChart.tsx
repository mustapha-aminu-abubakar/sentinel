'use client'

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { Skeleton } from '@/components/ui/skeleton'
import type { LatencyPoint } from '@/lib/api/types'

interface LatencyChartProps {
  data: LatencyPoint[]
  loading: boolean
}

function formatBucket(bucket: string): string {
  const d = new Date(bucket)
  if (isNaN(d.getTime())) return bucket
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function LatencyChart({ data, loading }: LatencyChartProps) {
  if (loading) {
    return <Skeleton className="h-[300px] w-full" />
  }

  if (data.length === 0) {
    return (
      <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
        No latency data for the selected period.
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="bucket"
          tickFormatter={(v) => formatBucket(String(v))}
          className="text-xs text-muted-foreground"
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          className="text-xs text-muted-foreground"
          tickLine={false}
          axisLine={false}
          unit="ms"
        />
        <Tooltip
          labelFormatter={(label) => formatBucket(String(label))}
          // Return [formattedValue, seriesName] tuple so Recharts renders both
          formatter={(value, name) => [`${Number(value).toFixed(1)} ms`, name]}
          contentStyle={{
            backgroundColor: 'var(--background)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
          }}
        />
        <Legend />
        <Line
          dataKey="avgLatencyMs"
          name="Avg Latency"
          stroke="var(--chart-1)"
          strokeWidth={2}
          dot={false}
        />
        <Line
          dataKey="p95LatencyMs"
          name="P95 Latency"
          stroke="var(--chart-3)"
          strokeWidth={2}
          strokeDasharray="4 4"
          dot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  )
}
