"use client"

import * as React from "react"
import { Loader2, Trash2 } from "lucide-react"

import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { BackendRequestError } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import { contentHubService, type ContentPillar } from "@/services/content-hubs.service"

/** Confirms deleting a pillar with a reason. Its sub-pillars and their assignments go with it; the assigned resources are kept. */
export function DeletePillarDialog({ hubId, live, pillar, pillars, onOpenChange, onDeleted }: { hubId: string; live: boolean; pillar: ContentPillar | null; pillars: ContentPillar[]; onOpenChange: (open: boolean) => void; onDeleted: () => void }) {
  const [reason, setReason] = React.useState("")
  const [deleting, setDeleting] = React.useState(false)
  const [error, setError] = React.useState("")

  React.useEffect(() => { setReason(""); setError("") }, [pillar?.id])

  const subPillars = React.useMemo(() => (pillar ? descendants(pillar.id, pillars) : []), [pillar, pillars])
  const assignments = [pillar, ...subPillars].reduce((total, row) => total + (row?.items?.length || 0), 0)

  async function remove() {
    if (!pillar) return
    setDeleting(true); setError("")
    try {
      await contentHubService.deletePillar(hubId, pillar.id, pillar.lock_version, reason.trim())
      showToast.success("Pillar deleted", "The reason is recorded in the audit history.")
      onOpenChange(false)
      onDeleted()
    } catch (value) {
      setError(value instanceof BackendRequestError && value.status === 409 ? "Someone changed this pillar after the page loaded. Close this, refresh the page and try again." : value instanceof Error ? value.message : "Unable to delete the pillar")
    } finally { setDeleting(false) }
  }

  const removed = [count(subPillars.length, "sub-pillar"), count(assignments, "resource assignment")].filter(Boolean)
  return <AlertDialog open={pillar !== null} onOpenChange={open => { if (!deleting) onOpenChange(open) }}>
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete “{pillar?.name || "Untitled"}”?</AlertDialogTitle>
        <AlertDialogDescription asChild>
          <div className="space-y-2 text-sm">
            {live && pillar?.status === "active" ? <p className="font-medium text-destructive">This hub is published. Deleting the pillar removes it from the MediGuide app and website straight away.</p> : null}
            <p><span className="font-medium text-foreground">Deleted with it:</span> {removed.length ? `${removed.join(" and ")}${subPillars.length ? ` (${subPillars.map(row => row.name).join(", ")})` : ""}.` : "nothing else; it has no sub-pillars or resource assignments."}</p>
            <p><span className="font-medium text-foreground">Kept:</span> the guidelines and other resources it points to. Only their place in this hub is removed.</p>
            <p>This can&apos;t be undone from the dashboard. To hide the pillar for now instead, archive it.</p>
          </div>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="space-y-2">
        <Label htmlFor="delete-pillar-reason">Reason</Label>
        <Textarea id="delete-pillar-reason" rows={3} value={reason} onChange={event => setReason(event.target.value)} placeholder="For example: merged into Clinical Management" disabled={deleting} />
      </div>
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      <AlertDialogFooter>
        <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
        <Button variant="destructive" disabled={deleting || !reason.trim()} onClick={() => void remove()}>
          {deleting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}Delete pillar
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
}

function descendants(id: string, pillars: ContentPillar[]): ContentPillar[] {
  const children = pillars.filter(row => row.parent_id === id)
  return children.flatMap(child => [child, ...descendants(child.id, pillars)])
}

function count(value: number, noun: string) {
  return value ? `${value} ${noun}${value === 1 ? "" : "s"}` : ""
}
