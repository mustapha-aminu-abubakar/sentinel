"use client"

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { createRule, updateRule } from '@/lib/api/sentinel'
import { validateRuleInput } from '@/lib/limits/validate'
import type { Client, RateRule } from '@/lib/api/types'

interface RuleFormDialogProps {
  mode: 'create' | 'edit'
  rule?: RateRule
  clients: Client[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export default function RuleFormDialog({
  mode,
  rule,
  clients,
  open,
  onOpenChange,
  onSuccess,
}: RuleFormDialogProps) {
  const [clientId, setClientId] = useState('')
  const [api, setApi] = useState('')
  const [requestsAllowed, setRequestsAllowed] = useState(100)
  const [windowSeconds, setWindowSeconds] = useState(60)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      if (mode === 'edit' && rule) {
        setClientId(rule.clientId)
        setApi(rule.api)
        setRequestsAllowed(rule.requestsAllowed)
        setWindowSeconds(rule.windowSeconds)
      } else {
        setClientId('')
        setApi('')
        setRequestsAllowed(100)
        setWindowSeconds(60)
      }
      setErrors({})
      setFormError(null)
      setSubmitting(false)
    }
  }, [open, mode, rule])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const input = {
      clientId,
      api: api.trim(),
      requestsAllowed,
      windowSeconds,
    }
    const validation = validateRuleInput(input)
    if (!validation.ok) {
      setErrors(validation.errors)
      return
    }
    setErrors({})
    setFormError(null)
    setSubmitting(true)
    try {
      if (mode === 'create') {
        await createRule(input)
      } else if (rule) {
        const patch: Record<string, unknown> = {}
        if (api.trim() !== rule.api) patch.api = api.trim()
        if (requestsAllowed !== rule.requestsAllowed)
          patch.requestsAllowed = requestsAllowed
        if (windowSeconds !== rule.windowSeconds)
          patch.windowSeconds = windowSeconds
        // Skip the API call if nothing changed
        if (Object.keys(patch).length > 0) {
          await updateRule(rule.id, patch)
        }
      }
      onSuccess()
      onOpenChange(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save rule')
    } finally {
      setSubmitting(false)
    }
  }

  const activeClients = clients.filter((c) => c.status === 'active')

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onOpenChange(next)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {mode === 'create' ? 'New Rule' : 'Edit Rule'}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {formError && (
            <div
              role="alert"
              className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {formError}
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="clientId">Client</Label>
            <Select
              value={clientId}
              onValueChange={(value) => {
                if (value) setClientId(value)
              }}
              disabled={mode === 'edit'}
            >
              <SelectTrigger className="w-full" id="clientId">
                <SelectValue placeholder="Select a client" />
              </SelectTrigger>
              <SelectContent>
                {activeClients.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.clientId && (
              <p id="clientId-error" className="text-xs text-destructive">{errors.clientId}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="api">API Identifier</Label>
            <Input
              id="api"
              value={api}
              onChange={(e) => setApi(e.target.value)}
              placeholder="e.g. stripe, openai"
              aria-invalid={!!errors.api}
              aria-describedby={errors.api ? 'api-error' : undefined}
            />
            {errors.api && (
              <p id="api-error" className="text-xs text-destructive">{errors.api}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="requestsAllowed">
                Requests Allowed
              </Label>
              <Input
                id="requestsAllowed"
                type="number"
                min={1}
                step={1}
                value={requestsAllowed}
                onChange={(e) =>
                  setRequestsAllowed(parseInt(e.target.value, 10) || 0)
                }
                aria-invalid={!!errors.requestsAllowed}
                aria-describedby={errors.requestsAllowed ? 'requestsAllowed-error' : undefined}
              />
              {errors.requestsAllowed && (
                <p id="requestsAllowed-error" className="text-xs text-destructive">
                  {errors.requestsAllowed}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="windowSeconds">
                Window (seconds)
              </Label>
              <Input
                id="windowSeconds"
                type="number"
                min={1}
                step={1}
                value={windowSeconds}
                onChange={(e) =>
                  setWindowSeconds(parseInt(e.target.value, 10) || 0)
                }
                aria-invalid={!!errors.windowSeconds}
                aria-describedby={errors.windowSeconds ? 'windowSeconds-error' : undefined}
              />
              {errors.windowSeconds && (
                <p id="windowSeconds-error" className="text-xs text-destructive">
                  {errors.windowSeconds}
                </p>
              )}
            </div>
          </div>

          <DialogFooter showCloseButton>
            <Button type="submit" disabled={submitting}>
              {submitting
                ? 'Saving...'
                : mode === 'create'
                  ? 'Create Rule'
                  : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
