"use client"

import * as React from "react"
import { Loader2, Trash2 } from "lucide-react"

import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { BackendRequestError } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import { formatStatus } from "@/lib/utils"
import { situationReportsService, type SituationReportRecord } from "@/services/outbreaks.service"

/** Confirms deleting a situation report. Its attachments and open corrections go with it. */
export function DeleteReportDialog({ report, onOpenChange, onDeleted }: { report: SituationReportRecord | null; onOpenChange: (open: boolean) => void; onDeleted: () => void }) {
  const [reason, setReason] = React.useState("")
  const [deleting, setDeleting] = React.useState(false)
  const [error, setError] = React.useState("")

  React.useEffect(() => { setReason(""); setError("") }, [report?.id])

  async function remove() {
    if (!report?.id || !report.lock_version) return
    setDeleting(true); setError("")
    try {
      await situationReportsService.remove(report.id, report.lock_version, reason.trim())
      showToast.success("Situation report deleted")
      onOpenChange(false)
      onDeleted()
    } catch (value) {
      setError(value instanceof BackendRequestError && value.status === 409 ? "Someone changed this report after the list loaded. Close this, refresh the list and try again." : value instanceof Error ? value.message : "Unable to delete the situation report")
    } finally { setDeleting(false) }
  }

  const live = report?.status === "published"
  return <AlertDialog open={report !== null} onOpenChange={open => { if (!deleting) onOpenChange(open) }}>
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete “{report?.title || "Untitled"}”?</AlertDialogTitle>
        <AlertDialogDescription asChild>
          <div className="space-y-2 text-sm">
            {live ? <p className="font-medium text-destructive">This report is published. Deleting it removes it from the MediGuide app straight away.</p> : null}
            {report?.status === "withdrawn" ? <p className="font-medium text-destructive">This report is withdrawn. Deleting it removes the record from the dashboard.</p> : null}
            <p><span className="font-medium text-foreground">Deleted with it:</span> its attachments and any open corrections.</p>
            <p>Status: {formatStatus(report?.status)}. The deletion and your reason are kept in the audit history.</p>
          </div>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="space-y-2">
        <Label htmlFor="delete-report-reason">Reason</Label>
        <Textarea id="delete-report-reason" rows={3} value={reason} onChange={event => setReason(event.target.value)} placeholder="For example: trial report entered by mistake" disabled={deleting} />
      </div>
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      <AlertDialogFooter>
        <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
        <Button variant="destructive" disabled={deleting || !reason.trim()} onClick={() => void remove()}>
          {deleting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}Delete report
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
}
