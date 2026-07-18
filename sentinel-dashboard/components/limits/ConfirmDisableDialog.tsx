"use client"

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { updateClient } from '@/lib/api/sentinel'
import type { Client } from '@/lib/api/types'

interface ConfirmDisableDialogProps {
  // client may be undefined when dialog is first mounted (before any client is selected)
  client: Client | undefined
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export default function ConfirmDisableDialog({
  client,
  open,
  onOpenChange,
  onSuccess,
}: ConfirmDisableDialogProps) {
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reset stale error whenever the dialog opens
  useEffect(() => {
    if (open) setError(null)
  }, [open])

  async function handleConfirm() {
    if (!client) return
    setSubmitting(true)
    setError(null)
    try {
      await updateClient(client.id, { status: 'inactive' })
      onSuccess()
      onOpenChange(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to disable client')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Disable {client?.name ?? ''}?</DialogTitle>
          <DialogDescription>
            This will deactivate the client and block all future requests.
            You can re-enable the client later by updating its status.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        )}
        <DialogFooter showCloseButton>
          <Button
            type="button"
            variant="destructive"
            onClick={handleConfirm}
            disabled={submitting || !client}
          >
            {submitting ? 'Disabling...' : 'Disable'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
