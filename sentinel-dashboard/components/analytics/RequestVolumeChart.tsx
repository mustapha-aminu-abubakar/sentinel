'use client'

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { Skeleton } from '@/components/ui/skeleton'
import type { UsagePoint } from '@/lib/api/types'
import type { Range } from '@/lib/analytics/range'
import { formatBucket, tickCount } from '@/lib/analytics/formatBucket'

interface RequestVolumeChartProps {
  data: UsagePoint[]
  loading: boolean
  statusFilter?: 'allowed' | 'rejected'
  range?: Range
}

export function RequestVolumeChart({
  data,
  loading,
  statusFilter,
  range,
}: RequestVolumeChartProps) {
  if (loading) {
    return <Skeleton className="h-[300px] w-full" />
  }

  if (data.length === 0) {
    return (
      <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
        No usage data for the selected period.
      </div>
    )
  }

  const showAllowed = !statusFilter || statusFilter === 'allowed'
  const showRejected = !statusFilter || statusFilter === 'rejected'

  return (
    <ResponsiveContainer width="100%" height={300}>
      <AreaChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
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
        <YAxis className="text-xs text-muted-foreground" tickLine={false} axisLine={false} />
        <Tooltip
          labelFormatter={(label) => formatBucket(String(label), range)}
          formatter={(value, name) => [
            Number(value).toLocaleString(),
            typeof name === 'string'
              ? name.charAt(0).toUpperCase() + name.slice(1)
              : name,
          ]}
          contentStyle={{
            backgroundColor: 'var(--background)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius)',
          }}
        />
        <Legend />
        {showAllowed && (
          <Area
            dataKey="allowed"
            name="Allowed"
            stackId="1"
            stroke="var(--chart-2)"
            fill="var(--chart-2)"
            fillOpacity={0.3}
          />
        )}
        {showRejected && (
          <Area
            dataKey="rejected"
            name="Rejected"
            stackId="1"
            stroke="var(--chart-5)"
            fill="var(--chart-5)"
            fillOpacity={0.3}
          />
        )}
      </AreaChart>
    </ResponsiveContainer>
  )
}
