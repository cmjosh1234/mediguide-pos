"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { showToast } from "@/lib/toast"
import { suggestNextGuidelineVersion } from "@/lib/guideline-version"
import { GuidelineDocumentRecord, GuidelineDocumentsService, GuidelineVersionRecord, isPublishedAsLink, isPublishedAsUploaded } from "@/services/guideline-documents.service"

export function GuidelineRevisionAction({ document, onChanged }: { document: GuidelineDocumentRecord; onChanged: () => Promise<unknown> }) {
  const router = useRouter()
  const [open, setOpen] = React.useState(false)
  const [version, setVersion] = React.useState("")
  const [busy, setBusy] = React.useState(false)
  const [created, setCreated] = React.useState<GuidelineVersionRecord | null>(null)
  const [error, setError] = React.useState("")
  const existing = document.versions.find(item => item.source_version_id === document.withdrawn_version_id && item.status !== "published")
  if (!document.withdrawn_at || !document.withdrawn_version_id) return null
  const asFile = isPublishedAsUploaded(document)
  const asLink = isPublishedAsLink(document)
  const goToDraft = (draft: GuidelineVersionRecord) => router.push(asFile || asLink
    ? `/guidelines/${document.id}#version-${draft.id}`
    : `/guidelines/${document.id}/versions/${draft.id}/markdown`)
  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (busy || !version.trim()) return
    if (document.versions.some(item => item.version.trim().toLowerCase() === version.trim().toLowerCase())) {
      setError("This version number already exists. Choose a different version number.")
      return
    }
    setBusy(true); setError("")
    try {
      const result = await GuidelineDocumentsService.reviseDocument(document.id, document.withdrawn_version_id!, version.trim())
      setCreated(result.version)
      await onChanged()
      showToast.success(result.existing ? "Continue existing draft" : "Revision draft created", "The document remains withdrawn until a version is published.")
    } catch (error) {
      setError(error instanceof Error ? error.message : "Could not create draft. Please try again.")
    } finally { setBusy(false) }
  }
  return <>
    <Button variant="outline" onClick={() => {
      if (existing) { goToDraft(existing); return }
      const source = document.versions.find(item => item.id === document.withdrawn_version_id)
      setOpen(true); setVersion(suggestNextGuidelineVersion(source?.version, document.versions.map(item => item.version))); setCreated(null); setError("")
    }}>{existing ? "Continue draft" : "Revise document"}</Button>
    <Dialog open={open} onOpenChange={value => { if (!busy) setOpen(value) }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{created ? "Revision draft ready" : "Revise withdrawn document"}</DialogTitle>
          <DialogDescription>{asFile
            ? "Create a new version and upload a replacement file. The withdrawn file stays in history."
            : asLink ? "Copy the withdrawn URL into a new editable draft. Review it before publication."
            : "Copy the published Markdown, or create Markdown from reviewed content if it is unavailable. Omitted content will be listed. All content in the new draft requires review."} The document stays withdrawn while you work.</DialogDescription>
        </DialogHeader>
        {created ? <div className="space-y-3">
          <p>Version {created.version} · Draft</p>
          {!!created.revision_warnings?.length && <ul className="list-disc space-y-2 pl-5 text-sm">{created.revision_warnings.map((warning, i) => <li key={i}>{warning}</li>)}</ul>}
          <Button onClick={() => { setOpen(false); goToDraft(created) }}>Open draft</Button>
        </div> : <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2"><Label htmlFor="revision-version">New version number</Label><Input id="revision-version" required maxLength={100} value={version} onChange={event => { setVersion(event.target.value); setError("") }} disabled={busy} aria-describedby="revision-version-help" /><p id="revision-version-help" className="text-sm text-muted-foreground">Suggested from the withdrawn version when its format is numeric. You can change it to match the document’s official version label.</p></div>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <DialogFooter><Button type="button" variant="outline" disabled={busy} onClick={() => setOpen(false)}>Cancel</Button><Button type="submit" disabled={busy || !version.trim()}>{busy ? "Creating…" : "Create revision draft"}</Button></DialogFooter>
        </form>}
      </DialogContent>
    </Dialog>
  </>
}
