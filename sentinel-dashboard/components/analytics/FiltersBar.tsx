'use client'

import { useMemo } from 'react'
import useSWR from 'swr'
import { listClients, listRules, sentinelKeys } from '@/lib/api/sentinel'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { Range } from '@/lib/analytics/range'

export interface FiltersState {
  clientId: string
  api: string
  range: Range
  status: string
}

interface FiltersBarProps {
  filters: FiltersState
  onFilterChange: (patch: Partial<FiltersState>) => void
}

export function FiltersBar({ filters, onFilterChange }: FiltersBarProps) {
  const {
    data: clients,
    isLoading: clientsLoading,
    error: clientsError,
  } = useSWR(sentinelKeys.clients(), () => listClients())

  const rulesParams = filters.clientId ? { clientId: filters.clientId } : undefined
  const { data: rules } = useSWR(
    sentinelKeys.rules(rulesParams),
    () => listRules(rulesParams)
  )

  const apiOptions = useMemo(() => {
    if (!rules) return []
    const apis = [...new Set(rules.map((r) => r.api))]
    apis.sort((a, b) => a.localeCompare(b))
    return apis
  }, [rules])

  const rangeOptions: { value: Range; label: string }[] = [
    { value: '1d', label: '1d' },
    { value: '7d', label: '7d' },
    { value: '30d', label: '30d' },
  ]

  return (
    <div className="flex flex-wrap items-end gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filter-client">Client</Label>
        <Select
          value={filters.clientId}
          onValueChange={(value) => onFilterChange({ clientId: value ?? '' })}
        >
          <SelectTrigger className="w-44" id="filter-client">
            {clientsLoading ? (
              <span className="text-muted-foreground">Loading…</span>
            ) : clientsError ? (
              <span className="text-destructive">Error</span>
            ) : (
              <SelectValue placeholder="All Clients" />
            )}
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Clients</SelectItem>
            {(clients ?? []).map((c) => (
              <SelectItem key={c.id} value={c.id}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filter-api">API</Label>
        <Select
          value={filters.api}
          onValueChange={(value) => onFilterChange({ api: value ?? '' })}
        >
          <SelectTrigger className="w-44" id="filter-api">
            <SelectValue placeholder="All APIs" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All APIs</SelectItem>
            {apiOptions.map((api) => (
              <SelectItem key={api} value={api}>
                {api}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label>Range</Label>
        <Tabs
          value={filters.range}
          onValueChange={(value) =>
            onFilterChange({
              range: value as Range,
            })
          }
        >
          <TabsList>
            {rangeOptions.map((opt) => (
              <TabsTrigger key={opt.value} value={opt.value}>
                {opt.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="filter-status">Status</Label>
        <Select
          value={filters.status}
          onValueChange={(value) => onFilterChange({ status: value ?? 'all' })}
        >
          <SelectTrigger className="w-32" id="filter-status">
            <SelectValue placeholder="All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="allowed">Allowed</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
