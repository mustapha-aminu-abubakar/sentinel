'use client'

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { Skeleton } from '@/components/ui/skeleton'
import type { LatencyPoint } from '@/lib/api/types'
import type { Range } from '@/lib/analytics/range'
import { formatBucket, tickCount } from '@/lib/analytics/formatBucket'

interface LatencyChartProps {
  data: LatencyPoint[]
  loading: boolean
  range?: Range
}

export function LatencyChart({ data, loading, range }: LatencyChartProps) {
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
      <BarChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="bucket"
          tickFormatter={(v) => formatBucket(String(v), range)}
          tickCount={tickCount(range)}
          domain={['dataMin', 'dataMax']}
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
          labelFormatter={(label) => formatBucket(String(label), range)}
          // Return [formattedValue, seriesName] tuple so Recharts renders both
          formatter={(value, name) => [`${Number(value).toFixed(1)} ms`, name]}
          contentStyle={{
            backgroundColor: 'var(--background)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
          }}
        />
        <Legend />
        <Bar
          dataKey="avgLatencyMs"
          name="Avg Latency"
          fill="var(--chart-1)"
          radius={[4, 4, 0, 0]}
        />
        <Bar
          dataKey="p95LatencyMs"
          name="P95 Latency"
          fill="var(--chart-3)"
          fillOpacity={0.7}
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  )
}
