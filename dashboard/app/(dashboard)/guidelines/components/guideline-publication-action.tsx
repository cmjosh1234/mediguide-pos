"use client"

import * as React from "react"
import { EyeOff, Send } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { showToast } from "@/lib/toast"
import { GuidelineDocumentRecord, GuidelineDocumentsService } from "@/services/guideline-documents.service"

export function GuidelinePublicationAction({ document, onChanged }: {
  document: GuidelineDocumentRecord
  onChanged: () => Promise<unknown>
}) {
  const [open, setOpen] = React.useState(false)
  const [reason, setReason] = React.useState("")
  const [reviewed, setReviewed] = React.useState(false)
  const [busy, setBusy] = React.useState(false)
  const withdrawn = Boolean(document.withdrawn_at)
  const versionId = withdrawn ? document.withdrawn_version_id : document.current_version_id
  const version = document.versions.find((item) => item.id === versionId)
  if (!versionId || version?.status !== "published") return null

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!versionId || busy) return
    setBusy(true)
    try {
      if (withdrawn) await GuidelineDocumentsService.publishVersion(versionId)
      else await GuidelineDocumentsService.withdrawDocument(document.id, versionId, reason.trim())
      await onChanged()
      setOpen(false)
      showToast.success(withdrawn ? "Document republished" : "Document withdrawn",
        withdrawn ? "The document is available on the website and app." : "Public access has been removed. Files and history are retained.")
    } catch (error) {
      showToast.error(withdrawn ? "Republish failed" : "Withdrawal failed", error instanceof Error ? error.message : "Please try again.")
    } finally {
      setBusy(false)
    }
  }

  return <>
    <Button variant="outline" size="sm" onClick={() => { setReason(""); setReviewed(false); setOpen(true) }}>
      {withdrawn ? <Send className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
      {withdrawn ? "Review and republish" : "Withdraw"}
    </Button>
    <Dialog open={open} onOpenChange={(value) => { if (!busy) setOpen(value) }}>
      <DialogContent>
        <form onSubmit={submit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{withdrawn ? "Review and republish document" : "Withdraw document"}</DialogTitle>
            <DialogDescription>
              {document.title} · Version {version.version}. {withdrawn
                ? "Republishing runs the publication checks and makes this approved version public again. To change its content, create and review a new version."
                : "Remove this document from the website, app, search and AI retrieval. It stays in the dashboard with its files and history. Downloaded copies are removed when devices next check online."}
            </DialogDescription>
          </DialogHeader>
          {withdrawn ? <div className="space-y-3 text-sm">
            <p className="whitespace-pre-wrap break-words"><strong>Withdrawal reason:</strong> {document.withdrawal_reason}</p>
            <p>Withdrawn {new Date(document.withdrawn_at!).toLocaleString()}</p>
            <a className="text-primary underline" href={`/guidelines/${document.id}`} target="_blank" rel="noopener noreferrer">Open document and version history for review</a>
            <label className="flex items-start gap-2">
              <input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} required disabled={busy} />
              I have reviewed this version and confirm it is ready for publication.
            </label>
          </div> : <div className="space-y-2">
            <Label htmlFor={`withdrawal-reason-${document.id}`}>Reason for withdrawal</Label>
            <Textarea id={`withdrawal-reason-${document.id}`} value={reason} onChange={(event) => setReason(event.target.value)} required maxLength={2000} disabled={busy} />
          </div>}
          <DialogFooter>
            <Button type="button" variant="outline" disabled={busy} onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit" variant={withdrawn ? "default" : "destructive"} disabled={busy || (withdrawn ? !reviewed : !reason.trim())}>
              {busy ? "Processing…" : withdrawn ? "Republish" : "Withdraw document"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </>
}
