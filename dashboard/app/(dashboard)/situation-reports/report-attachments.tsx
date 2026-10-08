"use client"

import * as React from "react"
import Link from "next/link"
import { AlertCircle, Eye, Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { showToast } from "@/lib/toast"
import type { DocumentKind } from "@/services/document-kinds.service"
import { outbreaksService } from "@/services/outbreaks.service"
import {
  situationReportsService,
  type SituationReportAttachmentInput,
  type SituationReportAttachmentRecord,
} from "@/services/situation-reports.service"
import {
  emptyResourceDraft,
  findResourceProblems,
  ResourceFields,
  type ResourceDraft,
} from "../outbreaks/resource-fields"

// The document type made for report attachments (migration 00057), kept as
// the uploaded file.
const ATTACHMENT_KIND = "situation_report_attachment"

function draftFromAttachment(attachment: SituationReportAttachmentRecord): ResourceDraft {
  return {
    title: attachment.title || "",
    description: attachment.description || "",
    issuing_organization: attachment.issuing_organization || "",
    document_type: attachment.document_kind || "",
    url: attachment.url || "",
  }
}

function attachmentPayload(draft: ResourceDraft): SituationReportAttachmentInput {
  return {
    title: draft.title,
    description: draft.description,
    issuing_organization: draft.issuing_organization,
    document_kind: draft.document_type,
    url: draft.url,
  }
}

const documentId = (url?: string) => url?.match(/^\/public\/guidelines\/([0-9a-fA-F-]{36})$/)?.[1]

/**
 * Documents from the guideline library attached to a situation report. They
 * are reviewed and published with the report, so they lock once it is
 * published; a correction starts with a copy of them.
 */
export function ReportAttachments({
  reportId,
  attachments,
  documentKinds,
  locked,
  hasUploadedPdf,
  onChanged,
}: {
  reportId: string
  attachments: SituationReportAttachmentRecord[]
  documentKinds: DocumentKind[]
  locked: boolean
  hasUploadedPdf: boolean
  onChanged: () => Promise<void>
}) {
  const defaultKind = documentKinds.some(kind => kind.slug === ATTACHMENT_KIND && kind.status === "active") ? ATTACHMENT_KIND : ""
  const emptyDraft = React.useCallback(() => ({ ...emptyResourceDraft(), document_type: defaultKind }), [defaultKind])
  const [draft, setDraft] = React.useState<ResourceDraft>(emptyDraft)
  const [attempted, setAttempted] = React.useState(false)
  const [busy, setBusy] = React.useState(false)
  const [editing, setEditing] = React.useState<SituationReportAttachmentRecord | null>(null)
  const { titles, loaded } = usePublishedTitles(attachments)
  const problems = findResourceProblems(draft, "attachment")
  const errorFor = (id: string) => attempted ? problems.find(problem => problem.id === id)?.message : undefined

  async function add() {
    if (problems.length > 0) {
      setAttempted(true)
      showToast.error("Missing required information", `Fill in: ${problems.map(problem => problem.label).join(", ")}`)
      return
    }
    setBusy(true)
    try {
      await situationReportsService.createAttachment(reportId, attachmentPayload(draft))
      setDraft(emptyDraft())
      setAttempted(false)
      showToast.success("Attachment added", "It is published with the report.")
      await onChanged()
    } catch (value) {
      showToast.error("Attachment not added", errorMessage(value))
    } finally {
      setBusy(false)
    }
  }

  async function remove(attachment: SituationReportAttachmentRecord) {
    if (!attachment.id || attachment.lock_version === undefined) return
    if (!window.confirm(`Remove "${attachment.title}" from this report?`)) return
    setBusy(true)
    try {
      await situationReportsService.removeAttachment(reportId, attachment.id, attachment.lock_version)
      showToast.success("Attachment removed", "The document itself stays in the guideline library.")
      await onChanged()
    } catch (value) {
      showToast.error("Attachment not removed", errorMessage(value))
    } finally {
      setBusy(false)
    }
  }

  return <Card>
    <CardHeader>
      <CardTitle>Attachments</CardTitle>
      <p className="text-sm text-muted-foreground">
        Published documents from the guideline library, reviewed and published with this report.
        To attach a new file, <Link href="/guidelines/create" className="underline">upload it under Guidelines</Link> first.
      </p>
    </CardHeader>
    <CardContent className="space-y-3">
      {locked ? <p className="flex items-center gap-2 text-sm text-muted-foreground">
        <AlertCircle className="h-4 w-4 shrink-0" />
        Attachments are locked once the report is published. Create a correction to change them.
      </p> : <div className="grid items-end gap-2 md:grid-cols-2">
        <ResourceFields
          className="contents"
          idPrefix="attachment"
          draft={draft}
          onChange={setDraft}
          documentKinds={documentKinds}
          reports={[]}
          errorFor={errorFor}
          allowSituationReports={false}
        />
        <Button className="md:col-span-2" disabled={busy} onClick={() => void add()}>
          <Plus className="mr-2 h-4 w-4" />
          Add attachment
        </Button>
      </div>}
      {hasUploadedPdf ? <p className="text-sm text-muted-foreground">
        This report also has a PDF uploaded before attachments were introduced. It is still served with the report.
      </p> : null}
      {attachments.length === 0 ? <p className="text-sm text-muted-foreground">No attachments yet.</p> : attachments.map(attachment => {
        const id = documentId(attachment.url)
        const kind = documentKinds.find(value => value.slug === attachment.document_kind)?.name ?? attachment.document_kind
        const document = attachment.url ? titles[attachment.url] : undefined
        return <div key={attachment.id} className="flex flex-col gap-3 rounded-md border p-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="min-w-0">
            <div className="truncate font-medium">{attachment.title}</div>
            <div className="truncate text-xs text-muted-foreground">
              {kind}
              {document ? ` · ${document}` : null}
            </div>
            {loaded && !document ? <p className="text-xs text-amber-700 dark:text-amber-400">
              This document is no longer published. Edit or remove the attachment before publishing.
            </p> : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {id ? <Button asChild size="sm" variant="outline">
              <Link href={`/guidelines/${id}`} target="_blank"><Eye className="mr-2 h-4 w-4" />View document</Link>
            </Button> : null}
            <Button size="sm" variant="outline" disabled={busy || locked} onClick={() => setEditing(attachment)}>Edit</Button>
            <Button size="sm" variant="destructive" disabled={busy || locked} onClick={() => void remove(attachment)}>Remove</Button>
          </div>
        </div>
      })}
    </CardContent>
    <Dialog open={Boolean(editing)} onOpenChange={open => { if (!open) setEditing(null) }}>
      <DialogContent className="sm:max-w-2xl">
        {editing ? <AttachmentEditor
          key={`${editing.id}:${editing.lock_version}`}
          reportId={reportId}
          attachment={editing}
          documentKinds={documentKinds}
          onCancel={() => setEditing(null)}
          onSaved={async () => { setEditing(null); await onChanged() }}
        /> : null}
      </DialogContent>
    </Dialog>
  </Card>
}

function AttachmentEditor({ reportId, attachment, documentKinds, onCancel, onSaved }: {
  reportId: string
  attachment: SituationReportAttachmentRecord
  documentKinds: DocumentKind[]
  onCancel: () => void
  onSaved: () => Promise<void>
}) {
  const [draft, setDraft] = React.useState<ResourceDraft>(() => draftFromAttachment(attachment))
  const [attempted, setAttempted] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const problems = findResourceProblems(draft, "edit-attachment")
  const errorFor = (id: string) => attempted ? problems.find(problem => problem.id === id)?.message : undefined

  async function save() {
    setAttempted(true)
    if (problems.length > 0 || !attachment.id) return
    setSaving(true)
    try {
      await situationReportsService.updateAttachment(reportId, attachment.id, { ...attachmentPayload(draft), lock_version: attachment.lock_version })
      showToast.success("Attachment updated", "Your changes were saved.")
      await onSaved()
    } catch (value) {
      showToast.error("Changes not saved", errorMessage(value))
    } finally {
      setSaving(false)
    }
  }

  return <>
    <DialogHeader>
      <DialogTitle>Edit attachment</DialogTitle>
      <DialogDescription>Changes are reviewed and published with the report.</DialogDescription>
    </DialogHeader>
    <ResourceFields
      idPrefix="edit-attachment"
      draft={draft}
      onChange={setDraft}
      documentKinds={documentKinds}
      reports={[]}
      errorFor={errorFor}
      allowSituationReports={false}
    />
    <DialogFooter>
      <Button variant="outline" onClick={onCancel} disabled={saving}>Cancel</Button>
      <Button onClick={() => void save()} disabled={saving}>Save changes</Button>
    </DialogFooter>
  </>
}

// Titles of the published documents the attachments point to, keyed by link.
// Once loaded, a link with no title is no longer published.
function usePublishedTitles(attachments: SituationReportAttachmentRecord[]) {
  const kinds = Array.from(new Set(attachments.map(attachment => attachment.document_kind).filter((kind): kind is string => Boolean(kind)))).sort().join(",")
  const [state, setState] = React.useState<{ kinds: string; titles: Record<string, string> } | null>(null)
  React.useEffect(() => {
    let cancelled = false
    void Promise.all(kinds ? kinds.split(",").map(kind => outbreaksService.listPublishedGuidelines("", kind)) : [])
      .then(pages => {
        if (cancelled) return
        const titles: Record<string, string> = {}
        for (const page of pages) for (const document of page.items || []) titles[`/public/guidelines/${document.id}`] = document.title || "Untitled document"
        setState({ kinds, titles })
      })
      .catch(() => { if (!cancelled) setState(null) })
    return () => { cancelled = true }
  }, [kinds])
  return { titles: state?.titles ?? {}, loaded: state?.kinds === kinds }
}

function errorMessage(value: unknown) {
  const message = value instanceof Error ? value.message : "The action failed."
  return /conflict|modified|lock/i.test(message) ? "Another editor changed this attachment. Reload before retrying." : message
}
