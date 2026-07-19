"use client"

import { useCallback, useMemo, useState } from 'react'
import useSWR from 'swr'
import {
  PlusIcon,
  PencilIcon,
  BanIcon,
  AlertCircle,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from 'lucide-react'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import {
  listClients,
  listRules,
  sentinelKeys,
} from '@/lib/api/sentinel'
import type { Client, RateRule } from '@/lib/api/types'
import { ApiError } from '@/lib/api/types'

import RuleFormDialog from '@/components/limits/RuleFormDialog'
import ClientFormDialog from '@/components/limits/ClientFormDialog'
import ConfirmDisableDialog from '@/components/limits/ConfirmDisableDialog'

type SortKey = 'client' | 'status' | 'api' | 'requests' | 'window'
type SortDir = 'asc' | 'desc'

function SortIcon({ active, dir }: { active: boolean; dir?: SortDir }) {
  if (!active) return <ArrowUpDown className="h-3 w-3 text-muted-foreground" />
  return dir === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />
}

function SortableHead({
  label,
  sortKey,
  currentSort,
  onToggle,
}: {
  label: string
  sortKey: SortKey
  currentSort: { key: SortKey; dir: SortDir } | null
  onToggle: (key: SortKey) => void
}) {
  const active = currentSort?.key === sortKey
  return (
    <TableHead className="cursor-pointer select-none" onClick={() => onToggle(sortKey)}>
      <div className="flex items-center gap-1">
        {label}
        <SortIcon active={active} dir={currentSort?.dir} />
      </div>
    </TableHead>
  )
}

interface FlatRow {
  client: Client
  rule: RateRule | null
}

export default function LimitsPage() {
  const {
    data: clients,
    error: clientsError,
    isLoading: clientsLoading,
    mutate: mutateClients,
  } = useSWR(sentinelKeys.clients(), listClients)

  const {
    data: rules,
    error: rulesError,
    isLoading: rulesLoading,
    mutate: mutateRules,
  } = useSWR(sentinelKeys.rules(), listRules)

  const [ruleDialogMode, setRuleDialogMode] = useState<'create' | 'edit'>('create')
  const [editingRule, setEditingRule] = useState<RateRule | undefined>()
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)

  const [clientDialogOpen, setClientDialogOpen] = useState(false)

  const [disablingClient, setDisablingClient] = useState<Client | undefined>()
  const [disableDialogOpen, setDisableDialogOpen] = useState(false)

  const [sort, setSort] = useState<{ key: SortKey; dir: SortDir } | null>(null)

  const toggleSort = useCallback((key: SortKey) => {
    setSort((prev) => {
      if (!prev || prev.key !== key) return { key, dir: 'asc' }
      if (prev.dir === 'asc') return { key, dir: 'desc' }
      return null
    })
  }, [])

  function openCreateRule() {
    setRuleDialogMode('create')
    setEditingRule(undefined)
    setRuleDialogOpen(true)
  }

  function openEditRule(rule: RateRule) {
    setRuleDialogMode('edit')
    setEditingRule(rule)
    setRuleDialogOpen(true)
  }

  function openDisableClient(client: Client) {
    setDisablingClient(client)
    setDisableDialogOpen(true)
  }

  const isLoading = clientsLoading || rulesLoading
  const error = clientsError || rulesError

  const rulesByClientId = useMemo(() => {
    const map: Record<string, RateRule[]> = {}
    for (const rule of rules ?? []) {
      if (!map[rule.clientId]) map[rule.clientId] = []
      map[rule.clientId].push(rule)
    }
    return map
  }, [rules])

  const clientsList = useMemo(() => clients ?? [], [clients])

  const flatRows: FlatRow[] = useMemo(() => {
    const rows: FlatRow[] = []
    for (const client of clientsList) {
      const clientRules = rulesByClientId[client.id] ?? []
      if (clientRules.length === 0) {
        rows.push({ client, rule: null })
      } else {
        for (const rule of clientRules) {
          rows.push({ client, rule })
        }
      }
    }
    return rows
  }, [clientsList, rulesByClientId])

  const sortedRows: FlatRow[] = useMemo(() => {
    if (!sort) return flatRows
    return [...flatRows].sort((a, b) => {
      const dir = sort.dir === 'asc' ? 1 : -1
      switch (sort.key) {
        case 'client':
          return dir * a.client.name.localeCompare(b.client.name)
        case 'status':
          return dir * a.client.status.localeCompare(b.client.status)
        case 'api': {
          const apiA = a.rule?.api ?? ''
          const apiB = b.rule?.api ?? ''
          return dir * apiA.localeCompare(apiB)
        }
        case 'requests':
          return dir * ((a.rule?.requestsAllowed ?? 0) - (b.rule?.requestsAllowed ?? 0))
        case 'window':
          return dir * ((a.rule?.windowSeconds ?? 0) - (b.rule?.windowSeconds ?? 0))
      }
    })
  }, [flatRows, sort])

  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold tracking-tight">Limit Management</h1>
        <Card>
          <CardHeader>
            <CardTitle>
              <Skeleton className="h-5 w-32" />
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold tracking-tight">Limit Management</h1>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Failed to load data</AlertTitle>
          <AlertDescription>
            {error instanceof ApiError
              ? `${error.status}: ${error.message}`
              : 'Network error — is the API server running?'}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Limit Management</h1>
          <p className="text-muted-foreground">
            Create, update, and manage rate limits for clients.
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => setClientDialogOpen(true)} variant="outline">
            <PlusIcon />
            New Client
          </Button>
          <Button onClick={openCreateRule}>
            <PlusIcon />
            New Rule
          </Button>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <SortableHead label="Client" sortKey="client" currentSort={sort} onToggle={toggleSort} />
            <SortableHead label="Status" sortKey="status" currentSort={sort} onToggle={toggleSort} />
            <SortableHead label="API" sortKey="api" currentSort={sort} onToggle={toggleSort} />
            <SortableHead label="Requests" sortKey="requests" currentSort={sort} onToggle={toggleSort} />
            <SortableHead label="Window" sortKey="window" currentSort={sort} onToggle={toggleSort} />
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {clientsList.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={6}
                className="py-12 text-center text-muted-foreground"
              >
                No clients yet.{' '}
                <Button
                  variant="link"
                  className="p-0 text-sm"
                  onClick={() => setClientDialogOpen(true)}
                >
                  Create your first client
                </Button>
              </TableCell>
            </TableRow>
          ) : (
            sortedRows.map((row) => {
              const { client, rule } = row
              return (
                <TableRow key={rule?.id ?? client.id}>
                  <TableCell className="font-medium">{client.name}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        client.status === 'active' ? 'default' : 'secondary'
                      }
                    >
                      {client.status}
                    </Badge>
                  </TableCell>
                  {rule ? (
                    <>
                      <TableCell className="font-mono text-xs">
                        {rule.api}
                      </TableCell>
                      <TableCell>{rule.requestsAllowed.toLocaleString()}</TableCell>
                      <TableCell>{rule.windowSeconds}s</TableCell>
                    </>
                  ) : (
                    <TableCell colSpan={3} className="text-muted-foreground">
                      No rules configured
                    </TableCell>
                  )}
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      {rule && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => openEditRule(rule)}
                          aria-label={`Edit rule for ${rule.api}`}
                        >
                          <PencilIcon aria-hidden />
                        </Button>
                      )}
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        disabled={client.status === 'inactive'}
                        onClick={() => openDisableClient(client)}
                        aria-label={`Disable client ${client.name}`}
                      >
                        <BanIcon aria-hidden />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })
          )}
        </TableBody>
      </Table>

      <RuleFormDialog
        mode={ruleDialogMode}
        rule={editingRule}
        clients={clientsList}
        open={ruleDialogOpen}
        onOpenChange={setRuleDialogOpen}
        onSuccess={() => mutateRules()}
      />

      <ClientFormDialog
        open={clientDialogOpen}
        onOpenChange={setClientDialogOpen}
        onSuccess={() => mutateClients()}
      />

      <ConfirmDisableDialog
        client={disablingClient}
        open={disableDialogOpen}
        onOpenChange={setDisableDialogOpen}
        onSuccess={() => mutateClients()}
      />
    </div>
  )
}
