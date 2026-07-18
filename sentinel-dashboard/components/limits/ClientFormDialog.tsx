"use client"

import { useState } from 'react'
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
import { createClient } from '@/lib/api/sentinel'
import { validateClientInput } from '@/lib/limits/validate'

interface ClientFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export default function ClientFormDialog({
  open,
  onOpenChange,
  onSuccess,
}: ClientFormDialogProps) {
  const [name, setName] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)

  function reset() {
    setName('')
    setErrors({})
    setSubmitting(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const validation = validateClientInput({ name })
    if (!validation.ok) {
      setErrors(validation.errors)
      return
    }
    setErrors({})
    setSubmitting(true)
    try {
      await createClient({ name })
      reset()
      onSuccess()
      onOpenChange(false)
    } catch {
      setErrors({ name: 'Failed to create client' })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset()
        onOpenChange(next)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Client</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="clientName">Client Name</Label>
            <Input
              id="clientName"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Stripe Integration"
              aria-invalid={!!errors.name}
              aria-describedby={errors.name ? 'clientName-error' : undefined}
            />
            {errors.name && (
              <p id="clientName-error" className="text-xs text-destructive">
                {errors.name}
              </p>
            )}
          </div>
          <DialogFooter showCloseButton>
            <Button type="submit" disabled={submitting}>
              {submitting ? 'Creating...' : 'Create Client'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
