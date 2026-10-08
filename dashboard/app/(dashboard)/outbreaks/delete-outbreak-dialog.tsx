"use client"

import * as React from "react"
import { Loader2, Trash2 } from "lucide-react"

import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { BackendRequestError } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import { outbreaksService, type OutbreakDeleteResult, type OutbreakRecord } from "@/services/outbreaks.service"

/** Confirms deleting an outbreak. Its own updates and resources go with it; linked situation reports and content hubs are kept. */
export function DeleteOutbreakDialog({ outbreak, onOpenChange, onDeleted }: { outbreak: OutbreakRecord | null; onOpenChange: (open: boolean) => void; onDeleted: () => void }) {
  const [reason, setReason] = React.useState("")
  const [deleting, setDeleting] = React.useState(false)
  const [error, setError] = React.useState("")

  React.useEffect(() => { setReason(""); setError("") }, [outbreak?.id])

  async function remove() {
    if (!outbreak?.id || !outbreak.lock_version) return
    setDeleting(true); setError("")
    try {
      const result = await outbreaksService.delete(outbreak.id, outbreak.lock_version, reason.trim())
      showToast.success("Outbreak deleted", keptSummary(result))
      onOpenChange(false)
      onDeleted()
    } catch (value) {
      setError(value instanceof BackendRequestError && value.status === 409 ? "Someone changed this outbreak after the list loaded. Close this, refresh the list and try again." : value instanceof Error ? value.message : "Unable to delete the outbreak")
    } finally { setDeleting(false) }
  }

  const live = outbreak && !["draft", "pending_review"].includes(outbreak.status || "")
  return <AlertDialog open={outbreak !== null} onOpenChange={open => { if (!deleting) onOpenChange(open) }}>
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete “{outbreak?.title || "Untitled"}”?</AlertDialogTitle>
        <AlertDialogDescription asChild>
          <div className="space-y-2 text-sm">
            {live ? <p className="font-medium text-destructive">This outbreak is {outbreak?.status?.replace("_", " ")}. Deleting it removes it from the MediGuide app straight away.</p> : null}
            <p><span className="font-medium text-foreground">Deleted with it:</span> its updates, resources, open corrections and any alerts not yet sent.</p>
            <p><span className="font-medium text-foreground">Kept:</span> situation reports and content hubs linked to it. They are only unlinked, and each report stays as visible in the app as it is now.</p>
          </div>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="space-y-2">
        <Label htmlFor="delete-outbreak-reason">Reason</Label>
        <Textarea id="delete-outbreak-reason" rows={3} value={reason} onChange={event => setReason(event.target.value)} placeholder="For example: entered twice" disabled={deleting} />
      </div>
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      <AlertDialogFooter>
        <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
        <Button variant="destructive" disabled={deleting || !reason.trim()} onClick={() => void remove()}>
          {deleting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}Delete outbreak
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
}

function keptSummary(result: OutbreakDeleteResult) {
  const kept = [count(result.unlinked_reports, "situation report"), count(result.unlinked_hubs, "content hub")].filter(Boolean)
  return kept.length ? `Kept and unlinked ${kept.join(" and ")}.` : undefined
}

function count(value: number | undefined, noun: string) {
  return value ? `${value} ${noun}${value === 1 ? "" : "s"}` : ""
}
