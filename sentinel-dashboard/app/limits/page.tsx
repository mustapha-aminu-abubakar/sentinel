"use client"

import { useState } from 'react'
import useSWR from 'swr'
import { PlusIcon, PencilIcon, BanIcon } from 'lucide-react'

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

  const [ruleDialogMode, setRuleDialogMode] = useState<'create' | 'edit'>(
    'create'
  )
  const [editingRule, setEditingRule] = useState<RateRule | undefined>()
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)

  const [clientDialogOpen, setClientDialogOpen] = useState(false)

  const [disablingClient, setDisablingClient] = useState<Client | undefined>()
  const [disableDialogOpen, setDisableDialogOpen] = useState(false)

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

  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold tracking-tight">
          Limit Management
        </h1>
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold tracking-tight">
          Limit Management
        </h1>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          Failed to load data:{' '}
          {error instanceof ApiError
            ? `${error.status}: ${error.message}`
            : 'An unexpected error occurred'}
        </div>
      </div>
    )
  }

  const rulesByClientId: Record<string, RateRule[]> = {}
  for (const rule of rules ?? []) {
    if (!rulesByClientId[rule.clientId]) {
      rulesByClientId[rule.clientId] = []
    }
    rulesByClientId[rule.clientId].push(rule)
  }

  const clientsList = clients ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            Limit Management
          </h1>
          <p className="text-muted-foreground">
            Create, update, and manage rate limits for clients.
          </p>
        </div>
        <div className="flex gap-2">
          {/* New Client is always available — not gated on empty list */}
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
            <TableHead>Client</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>API</TableHead>
            <TableHead>Requests</TableHead>
            <TableHead>Window</TableHead>
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
            clientsList.map((client) => {
              const clientRules = rulesByClientId[client.id] ?? []
              return clientRules.length > 0
                ? clientRules.map((rule, idx) => (
                    <TableRow key={rule.id}>
                      {idx === 0 && (
                        <TableCell
                          rowSpan={clientRules.length}
                          className="font-medium"
                        >
                          {client.name}
                        </TableCell>
                      )}
                      {idx === 0 && (
                        <TableCell
                          rowSpan={clientRules.length}
                        >
                          <Badge
                            variant={
                              client.status === 'active'
                                ? 'default'
                                : 'secondary'
                            }
                          >
                            {client.status}
                          </Badge>
                        </TableCell>
                      )}
                      <TableCell className="font-mono text-xs">
                        {rule.api}
                      </TableCell>
                      <TableCell>
                        {rule.requestsAllowed.toLocaleString()}
                      </TableCell>
                      <TableCell>{rule.windowSeconds}s</TableCell>
                      {idx === 0 && (
                        <TableCell
                          rowSpan={clientRules.length}
                          className="text-right"
                        >
                          <div className="flex justify-end gap-1">
                            {clientRules.map((r) => (
                              <Button
                                key={r.id}
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => openEditRule(r)}
                                aria-label={`Edit rule for ${r.api}`}
                              >
                                <PencilIcon aria-hidden />
                              </Button>
                            ))}
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
                      )}
                    </TableRow>
                  ))
                : [
                    <TableRow key={client.id}>
                      <TableCell className="font-medium">
                        {client.name}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            client.status === 'active'
                              ? 'default'
                              : 'secondary'
                          }
                        >
                          {client.status}
                        </Badge>
                      </TableCell>
                      <TableCell
                        colSpan={3}
                        className="text-muted-foreground"
                      >
                        No rules configured
                      </TableCell>
                      <TableCell className="text-right">
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
                      </TableCell>
                    </TableRow>,
                  ]
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

      {/* Pass disablingClient directly — ConfirmDisableDialog handles undefined safely */}
      <ConfirmDisableDialog
        client={disablingClient}
        open={disableDialogOpen}
        onOpenChange={setDisableDialogOpen}
        onSuccess={() => mutateClients()}
      />
    </div>
  )
}
