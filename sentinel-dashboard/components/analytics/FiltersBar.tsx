'use client'

import useSWR from 'swr'
import { listClients, sentinelKeys } from '@/lib/api/sentinel'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { Range } from '@/lib/analytics/range'

export interface FiltersState {
  clientId: string
  api: string
  range: Range | 'custom'
  from: string
  to: string
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

  const rangeOptions: { value: Range | 'custom'; label: string }[] = [
    { value: '10d', label: '10d' },
    { value: '15d', label: '15d' },
    { value: '30d', label: '30d' },
    { value: 'custom', label: 'Custom' },
  ]

  const isCustom = filters.range === 'custom'

  function handleFromChange(value: string) {
    // Validate that from <= to when both are set
    if (value && filters.to && value > filters.to) return
    onFilterChange({ from: value })
  }

  function handleToChange(value: string) {
    // Validate that to >= from when both are set
    if (value && filters.from && value < filters.from) return
    onFilterChange({ to: value })
  }

  return (
    <div className="flex flex-wrap items-end gap-4">
      <div className="flex flex-col gap-1.5">
        <label
          htmlFor="filter-client"
          className="text-xs font-medium text-muted-foreground"
        >
          Client
        </label>
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
        <label
          htmlFor="filter-api"
          className="text-xs font-medium text-muted-foreground"
        >
          API
        </label>
        <Input
          id="filter-api"
          placeholder="All APIs"
          className="h-8 w-36"
          value={filters.api}
          onChange={(e) => onFilterChange({ api: e.target.value })}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <span
          id="filter-range-label"
          className="text-xs font-medium text-muted-foreground"
        >
          Range
        </span>
        <Tabs
          value={filters.range}
          aria-labelledby="filter-range-label"
          onValueChange={(value) =>
            onFilterChange({
              range: value as Range | 'custom',
              // Clear custom date params when switching to a preset range
              ...(value !== 'custom' && { from: '', to: '' }),
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

      {isCustom && (
        <>
          <div className="flex flex-col gap-1.5">
            <label
              htmlFor="filter-from"
              className="text-xs font-medium text-muted-foreground"
            >
              From
            </label>
            <Input
              id="filter-from"
              type="date"
              className="h-8 w-40"
              value={filters.from}
              max={filters.to || undefined}
              onChange={(e) => handleFromChange(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label
              htmlFor="filter-to"
              className="text-xs font-medium text-muted-foreground"
            >
              To
            </label>
            <Input
              id="filter-to"
              type="date"
              className="h-8 w-40"
              value={filters.to}
              min={filters.from || undefined}
              onChange={(e) => handleToChange(e.target.value)}
            />
          </div>
        </>
      )}

      <div className="flex flex-col gap-1.5">
        <label
          htmlFor="filter-status"
          className="text-xs font-medium text-muted-foreground"
        >
          Status
        </label>
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
