"use client"

import * as React from "react"
import Link from "next/link"
import { ExternalLink, Eye } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { showToast } from "@/lib/toast"
import type { DocumentKind } from "@/services/document-kinds.service"
import {
  outbreaksService,
  type OutbreakResourceRecord,
  type OutbreakUpdateRecord,
  type SituationReportRecord,
} from "@/services/outbreaks.service"
import {
  draftFromResource,
  Field,
  findResourceProblems,
  KIND_DESCRIPTION_OVERRIDES,
  ResourceFields,
  resourcePayload,
  SITUATION_REPORT_TYPE,
  type ResourceDraft,
} from "./resource-fields"
import { PaginationControls, usePagination } from "./list-pagination"

type ChildRecord = OutbreakUpdateRecord | OutbreakResourceRecord
type ChildKind = "update" | "resource"
type Action = "submit" | "approve" | "publish" | "withdraw"

const LEGACY_RESOURCE_TYPES: Record<string, { label: string; description: string }> = {
  situation_report: SITUATION_REPORT_TYPE,
  internal_route: { label: "Internal route", description: "A page within the app." },
  approved_external_url: { label: "Approved external URL", description: "A link to an approved external website." },
}

const SUPERSEDED_PREFIX = "superseded by approved"

function describeResourceType(item: OutbreakResourceRecord, kinds: DocumentKind[]) {
  if (item.resource_type === "guideline") {
    const kind = kinds.find(value => value.slug === item.document_kind)
    const slug = item.document_kind || "guideline"
    return {
      label: kind?.name ?? slug,
      description: KIND_DESCRIPTION_OVERRIDES[slug] ?? kind?.description ?? "",
    }
  }
  return item.resource_type ? LEGACY_RESOURCE_TYPES[item.resource_type] ?? { label: item.resource_type, description: "" } : undefined
}

function isRetired(item: ChildRecord) {
  return item.status === "withdrawn" && (item.withdrawal_reason || "").toLowerCase().startsWith(SUPERSEDED_PREFIX)
}

function isOpen(item: ChildRecord) {
  return item.status === "draft" || item.status === "pending_review"
}

type EditTarget = { item: ChildRecord; mode: "update" | "correct" }

type Gates = { approveBlocked?: string; publishBlocked?: string }

// Mirrors the backend rules for resources and updates (transitionChildWithHook).
function workflowGates(item: ChildRecord, userId: string, parentPublished: boolean, label: string): Gates {
  return {
    approveBlocked: userId && item.author_id === userId ? `You created this ${label}, so someone else has to approve it.` : undefined,
    publishBlocked: parentPublished ? undefined : `Publish the outbreak before publishing this ${label}.`,
  }
}

function GateNote({ item, gates }: { item: ChildRecord; gates: Gates }) {
  if (item.status !== "pending_review") return null
  const note = item.approved_at ? gates.publishBlocked : gates.approveBlocked
  return note ? <p className="w-full text-xs text-amber-700 lg:text-right dark:text-amber-400">{note}</p> : null
}

export function ChildContentWorkflow({
  outbreakId,
  kind,
  items,
  empty,
  onChanged,
  linkTitles = {},
  documentKinds = [],
  reports = [],
  pageSize,
  currentUserId = "",
  parentPublished = false,
}: {
  outbreakId: string
  kind: ChildKind
  items: ChildRecord[]
  empty: string
  onChanged: () => Promise<void>
  linkTitles?: Record<string, string>
  documentKinds?: DocumentKind[]
  reports?: SituationReportRecord[]
  pageSize?: number
  currentUserId?: string
  parentPublished?: boolean
}) {
  const [workingId, setWorkingId] = React.useState<string | null>(null)
  const [viewingId, setViewingId] = React.useState<string | null>(null)
  const [editing, setEditing] = React.useState<EditTarget | null>(null)
  const viewing = items.find(item => item.id === viewingId)
  const byId = new Map(items.map(item => [item.id, item]))

  // Edits hang off the version they replace; retired versions are history only.
  const pendingEdits = new Map<string, ChildRecord>()
  for (const item of items) {
    if (item.supersedes_id && isOpen(item) && byId.has(item.supersedes_id)) {
      const existing = pendingEdits.get(item.supersedes_id)
      if (!existing || (item.created_at || "") > (existing.created_at || "")) pendingEdits.set(item.supersedes_id, item)
    }
  }
  const nestedIds = new Set(Array.from(pendingEdits.values(), edit => edit.id))
  const rows = items.filter(item => !isRetired(item) && !nestedIds.has(item.id))
  const pagination = usePagination(rows, pageSize)
  const previousVersion = (item: ChildRecord) => {
    const previous = item.supersedes_id ? byId.get(item.supersedes_id) : undefined
    return previous && isRetired(previous) ? previous : undefined
  }

  async function run(item: ChildRecord, action: Action) {
    if (!item.id || item.lock_version === undefined) return

    let reason = ""
    if (action === "withdraw") {
      reason = window.prompt(`Enter the reason for this ${action}`)?.trim() || ""
      if (!reason) return
    }
    if (action === "publish" && !window.confirm(`Publish this outbreak ${kind}? Published content is immutable.`)) return

    setWorkingId(item.id)
    try {
      const input = { lock_version: item.lock_version, reason }
      if (kind === "update") await outbreaksService.transitionUpdate(outbreakId, item.id, action, input)
      else await outbreaksService.transitionResource(outbreakId, item.id, action, input)
      await onChanged()
      showToast.success("Workflow updated", `The ${kind} workflow action completed.`)
    } catch (value) {
      showToast.error("Workflow failed", errorMessage(value))
    } finally {
      setWorkingId(null)
    }
  }

  async function discard(edit: ChildRecord) {
    if (!edit.id || edit.lock_version === undefined) return
    if (!window.confirm("Discard this edit? The published version stays as it is.")) return
    setWorkingId(edit.id)
    try {
      if (kind === "update") await outbreaksService.deleteUpdate(outbreakId, edit.id, edit.lock_version)
      else await outbreaksService.deleteResource(outbreakId, edit.id, edit.lock_version)
      await onChanged()
      showToast.success("Edit discarded", "The published version is unchanged.")
    } catch (value) {
      showToast.error("Edit not discarded", errorMessage(value))
    } finally {
      setWorkingId(null)
    }
  }

  function openEditor(item: ChildRecord) {
    const pending = item.id ? pendingEdits.get(item.id) : undefined
    if (pending) setEditing({ item: pending, mode: "update" })
    else setEditing({ item, mode: item.status === "published" ? "correct" : "update" })
  }

  async function closeEditor() {
    setEditing(null)
    await onChanged()
  }

  if (!rows.length) return <p className="text-sm text-muted-foreground">{empty}</p>

  return <div className="space-y-2">
    {pagination.visible.map(item => {
      const status = item.status || "draft"
      const busy = workingId === item.id
      const isResource = "resource_type" in item
      const pending = item.id ? pendingEdits.get(item.id) : undefined
      const gates = workflowGates(item, currentUserId, parentPublished, kind)
      const detail = isResource ? describeResourceType(item, documentKinds)?.label : "summary" in item ? item.summary : undefined
      return <div key={item.id} className="rounded-md border p-3">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <TitleButton item={item} kind={kind} detail={detail} onOpen={() => setViewingId(item.id ?? null)} />
          <div className="flex flex-wrap items-center gap-2 lg:justify-end">
            <Badge variant="outline">{status}</Badge>
            <>
              {status === "draft" ? <Button size="sm" variant="outline" disabled={busy} onClick={() => void run(item, "submit")}>Submit</Button> : null}
              {status === "pending_review" && !item.approved_at ? <Button size="sm" variant="outline" disabled={busy || Boolean(gates.approveBlocked)} onClick={() => void run(item, "approve")}>Approve</Button> : null}
              {status === "pending_review" && item.approved_at ? <Button size="sm" disabled={busy || Boolean(gates.publishBlocked)} onClick={() => void run(item, "publish")}>Publish</Button> : null}
              {isOpen(item) || status === "published" ? <Button size="sm" variant="outline" disabled={busy} onClick={() => openEditor(item)}>{pending ? "Edit pending change" : "Edit"}</Button> : null}
              {status === "published" ? <Button size="sm" variant="destructive" disabled={busy} onClick={() => void run(item, "withdraw")}>Withdraw</Button> : null}
              <GateNote item={item} gates={gates} />
            </>
          </div>
        </div>
        {pending ? <PendingEditRow
          kind={kind}
          edit={pending}
          gates={workflowGates(pending, currentUserId, parentPublished, "edit")}
          busy={workingId === pending.id}
          onOpen={() => setViewingId(pending.id ?? null)}
          onRun={action => void run(pending, action)}
          onEdit={() => setEditing({ item: pending, mode: "update" })}
          onDiscard={() => void discard(pending)}
        /> : null}
      </div>
    })}
    <PaginationControls pagination={pagination} label={kind === "resource" ? "Resources" : "Updates"} />
    <Dialog open={Boolean(viewing)} onOpenChange={open => { if (!open) setViewingId(null) }}>
      <DialogContent className="sm:max-w-xl">
        {viewing ? <ChildDetails item={viewing} kind={kind} linkTitles={linkTitles} documentKinds={documentKinds} previous={previousVersion(viewing)} /> : null}
      </DialogContent>
    </Dialog>
    <Dialog open={Boolean(editing)} onOpenChange={open => { if (!open) setEditing(null) }}>
      <DialogContent className="sm:max-w-2xl">
        {editing ? "resource_type" in editing.item ? <ResourceEditor
          key={`${editing.item.id}:${editing.item.lock_version}`}
          outbreakId={outbreakId}
          resource={editing.item}
          mode={editing.mode}
          documentKinds={documentKinds}
          reports={reports}
          onCancel={() => setEditing(null)}
          onSaved={closeEditor}
        /> : <UpdateEditor
          key={`${editing.item.id}:${editing.item.lock_version}`}
          outbreakId={outbreakId}
          update={editing.item}
          mode={editing.mode}
          onCancel={() => setEditing(null)}
          onSaved={closeEditor}
        /> : null}
      </DialogContent>
    </Dialog>
  </div>
}

function TitleButton({ item, kind, detail, onOpen }: { item: ChildRecord; kind: ChildKind; detail?: string; onOpen: () => void }) {
  return <button
    type="button"
    className="min-w-0 rounded-sm text-left hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    onClick={onOpen}
    aria-label={`View details of ${item.title || `untitled ${kind}`}`}
  >
    <div className="truncate font-medium">{item.title || `Untitled ${kind}`}</div>
    {detail ? <div className="truncate text-xs text-muted-foreground">{detail}</div> : null}
  </button>
}

function PendingEditRow({ kind, edit, gates, busy, onOpen, onRun, onEdit, onDiscard }: {
  kind: ChildKind
  edit: ChildRecord
  gates: Gates
  busy: boolean
  onOpen: () => void
  onRun: (action: Action) => void
  onEdit: () => void
  onDiscard: () => void
}) {
  const inReview = edit.status === "pending_review"
  return <div className="mt-3 flex flex-col gap-3 border-l-2 border-amber-400 pl-3 lg:flex-row lg:items-center lg:justify-between">
    <div className="min-w-0">
      <Badge variant="secondary" className="mb-1">{inReview ? "Edit pending review" : "Edit draft"}</Badge>
      <TitleButton item={edit} kind={kind} detail="Replaces the published version once approved and published" onOpen={onOpen} />
    </div>
    <div className="flex flex-wrap items-center gap-2 lg:justify-end">
      {edit.status === "draft" ? <Button size="sm" variant="outline" disabled={busy} onClick={() => onRun("submit")}>Submit</Button> : null}
      {inReview && !edit.approved_at ? <Button size="sm" variant="outline" disabled={busy || Boolean(gates.approveBlocked)} onClick={() => onRun("approve")}>Approve</Button> : null}
      {inReview && edit.approved_at ? <Button size="sm" disabled={busy || Boolean(gates.publishBlocked)} onClick={() => onRun("publish")}>Publish</Button> : null}
      <Button size="sm" variant="outline" disabled={busy} onClick={onEdit}>Edit</Button>
      <Button size="sm" variant="destructive" disabled={busy} onClick={onDiscard}>Discard</Button>
      <GateNote item={edit} gates={gates} />
    </div>
  </div>
}

type EditorProps = {
  outbreakId: string
  mode: EditTarget["mode"]
  onCancel: () => void
  onSaved: () => Promise<void>
}

function editNote(kind: ChildKind, item: ChildRecord, mode: EditTarget["mode"]) {
  const isEdit = Boolean(item.supersedes_id)
  return mode === "correct"
    ? "The published version stays live until your edit is approved and published. Your edit is submitted for review when you save."
    : item.status === "pending_review"
      ? isEdit
        ? "This edit is waiting for review. Saving clears any approval so it is reviewed again."
        : `Saving sends this ${kind} back to draft so the changes can be reviewed.`
      : isEdit
        ? "Saving updates this pending edit."
        : "Changes are saved to this draft."
}

function editTitle(kind: ChildKind, item: ChildRecord, mode: EditTarget["mode"]) {
  return mode === "correct" ? `Edit published ${kind}` : item.supersedes_id ? "Edit pending change" : `Edit ${kind}`
}

function ReasonField({ idPrefix, value, onChange, showError }: { idPrefix: string; value: string; onChange: (value: string) => void; showError: boolean }) {
  return <div className="space-y-1">
    <Label htmlFor={`${idPrefix}-reason`}>Reason for this change<span className="text-destructive">*</span></Label>
    <Textarea
      id={`${idPrefix}-reason`}
      rows={2}
      maxLength={500}
      aria-required
      aria-invalid={showError}
      placeholder={idPrefix === "edit-resource" ? "For example: the linked guideline was replaced by a newer edition" : "For example: late reports changed the case count"}
      value={value}
      onChange={event => onChange(event.target.value)}
    />
    {showError ? <p className="text-xs text-destructive">A reason is required.</p> : null}
  </div>
}

function ResourceEditor({ outbreakId, resource, mode, documentKinds, reports, onCancel, onSaved }: EditorProps & {
  resource: OutbreakResourceRecord
  documentKinds: DocumentKind[]
  reports: SituationReportRecord[]
}) {
  const [draft, setDraft] = React.useState<ResourceDraft>(() => draftFromResource(resource))
  const [reason, setReason] = React.useState("")
  const [attempted, setAttempted] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const problems = findResourceProblems(draft, "edit-resource")
  const reasonMissing = mode === "correct" && !reason.trim()
  const errorFor = (id: string) => attempted ? problems.find(problem => problem.id === id)?.message : undefined
  const isEdit = Boolean(resource.supersedes_id)
  const needsNewType = !draftFromResource(resource).document_type

  async function save() {
    setAttempted(true)
    if (problems.length > 0 || reasonMissing || !resource.id || resource.lock_version === undefined) return
    setSaving(true)
    try {
      if (mode === "correct") {
        await outbreaksService.editPublishedResource(outbreakId, resource.id, {
          lock_version: resource.lock_version,
          reason: reason.trim(),
          changes: resourcePayload(draft),
        })
        showToast.success("Edit submitted for review", "The published version stays live until the edit is published.")
      } else {
        await outbreaksService.updateResource(outbreakId, resource.id, { ...resourcePayload(draft), lock_version: resource.lock_version })
        showToast.success("Resource updated", resource.status === "pending_review" && !isEdit ? "It is back in draft for review." : "Your changes were saved.")
      }
      await onSaved()
    } catch (value) {
      showToast.error("Changes not saved", errorMessage(value))
    } finally {
      setSaving(false)
    }
  }

  return <>
    <DialogHeader>
      <DialogTitle>{editTitle("resource", resource, mode)}</DialogTitle>
      <DialogDescription>{editNote("resource", resource, mode)}</DialogDescription>
    </DialogHeader>
    {needsNewType ? <p className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950">
      This resource uses an older link type. Choose a document type and a published document to replace it.
    </p> : null}
    <ResourceFields
      idPrefix="edit-resource"
      draft={draft}
      onChange={setDraft}
      documentKinds={documentKinds}
      reports={reports}
      errorFor={errorFor}
    />
    {mode === "correct" ? <ReasonField idPrefix="edit-resource" value={reason} onChange={setReason} showError={attempted && reasonMissing} /> : null}
    <DialogFooter>
      <Button variant="outline" onClick={onCancel} disabled={saving}>Cancel</Button>
      <Button onClick={() => void save()} disabled={saving}>{mode === "correct" ? "Submit edit for review" : "Save changes"}</Button>
    </DialogFooter>
  </>
}

function UpdateEditor({ outbreakId, update, mode, onCancel, onSaved }: EditorProps & { update: OutbreakUpdateRecord }) {
  const [title, setTitle] = React.useState(update.title || "")
  const [summary, setSummary] = React.useState(update.summary || "")
  const [reason, setReason] = React.useState("")
  const [attempted, setAttempted] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const titleMissing = !title.trim()
  const reasonMissing = mode === "correct" && !reason.trim()
  const isEdit = Boolean(update.supersedes_id)

  async function save() {
    setAttempted(true)
    if (titleMissing || reasonMissing || !update.id || update.lock_version === undefined) return
    setSaving(true)
    try {
      const changes = { title: title.trim(), summary: summary.trim() }
      if (mode === "correct") {
        await outbreaksService.editPublishedUpdate(outbreakId, update.id, {
          lock_version: update.lock_version,
          reason: reason.trim(),
          changes,
        })
        showToast.success("Edit submitted for review", "The published version stays live until the edit is published.")
      } else {
        await outbreaksService.updateUpdate(outbreakId, update.id, { ...changes, lock_version: update.lock_version })
        showToast.success("Update saved", update.status === "pending_review" && !isEdit ? "It is back in draft for review." : "Your changes were saved.")
      }
      await onSaved()
    } catch (value) {
      showToast.error("Changes not saved", errorMessage(value))
    } finally {
      setSaving(false)
    }
  }

  return <>
    <DialogHeader>
      <DialogTitle>{editTitle("update", update, mode)}</DialogTitle>
      <DialogDescription>{editNote("update", update, mode)}</DialogDescription>
    </DialogHeader>
    <div className="space-y-3">
      <Field id="edit-update-title" label="Title" required error={attempted && titleMissing ? "Title is required." : undefined}>
        <Input
          id="edit-update-title"
          maxLength={240}
          aria-required
          aria-invalid={attempted && titleMissing}
          value={title}
          onChange={event => setTitle(event.target.value)}
        />
      </Field>
      <Field id="edit-update-summary" label="Summary">
        <Textarea
          id="edit-update-summary"
          rows={5}
          maxLength={10000}
          value={summary}
          onChange={event => setSummary(event.target.value)}
        />
      </Field>
    </div>
    {mode === "correct" ? <ReasonField idPrefix="edit-update" value={reason} onChange={setReason} showError={attempted && reasonMissing} /> : null}
    <DialogFooter>
      <Button variant="outline" onClick={onCancel} disabled={saving}>Cancel</Button>
      <Button onClick={() => void save()} disabled={saving}>{mode === "correct" ? "Submit edit for review" : "Save changes"}</Button>
    </DialogFooter>
  </>
}

function ChildDetails({ item, kind, linkTitles, documentKinds, previous }: { item: ChildRecord; kind: ChildKind; linkTitles: Record<string, string>; documentKinds: DocumentKind[]; previous?: ChildRecord }) {
  const rows: Array<[string, React.ReactNode]> = []
  if ("resource_type" in item) {
    const type = describeResourceType(item, documentKinds)
    rows.push(["Document type", type ? <><span>{type.label}</span>{type.description ? <span className="block text-xs text-muted-foreground">{type.description}</span> : null}</> : undefined])
    if (item.url) {
      const linked = linkTitles[item.url]
      rows.push([
        item.resource_type === "guideline" ? "Linked document" : item.resource_type === "situation_report" ? "Linked situation report" : "Route or URL",
        linked ? linked : <span className="break-all">{item.url}</span>,
      ])
      const target = linkedTarget(item)
      if (target) rows.push(["Open", <LinkedDocumentActions key="open" target={target} />])
    }
    if (item.asset_url) rows.push(["Asset path", <span key="asset" className="break-all">{item.asset_url}</span>])
    rows.push(["Issuing organization", item.issuing_organization])
    rows.push(["Description", item.description])
    rows.push(["Sort order", item.sort_order])
  } else if ("summary" in item) {
    rows.push(["Summary", item.summary])
  }
  rows.push(["Status", <Badge key="status" variant="outline">{item.status || "draft"}</Badge>])
  rows.push(["Created", formatDate(item.created_at)])
  rows.push(["Last updated", formatDate(item.updated_at)])
  rows.push(["Approved", formatDate(item.approved_at)])
  rows.push(["Published", formatDate(item.published_at)])
  rows.push(["Withdrawn", formatDate(item.withdrawn_at)])
  rows.push(["Withdrawal reason", item.withdrawal_reason])
  if (previous) {
    const reason = (previous.withdrawal_reason || "").replace(/^superseded by approved (correction|revision):?\s*/i, "")
    rows.push(["Earlier version", `“${previous.title || "Untitled"}” replaced on ${formatDate(previous.withdrawn_at) ?? "an unknown date"}${reason ? ` — ${reason}` : ""}`])
  }

  return <>
    <DialogHeader>
      <DialogTitle className="pr-6">{item.title || `Untitled ${kind}`}</DialogTitle>
      <DialogDescription>{kind === "resource" ? "Typed resource details" : "Outbreak update details"}</DialogDescription>
    </DialogHeader>
    <dl className="grid gap-x-4 gap-y-3 text-sm sm:grid-cols-[10rem_1fr]">
      {rows.map(([label, value]) => <React.Fragment key={label}>
        <dt className="text-muted-foreground">{label}</dt>
        <dd className="whitespace-pre-wrap">{value === undefined || value === null || value === "" ? <span className="text-muted-foreground">—</span> : value}</dd>
      </React.Fragment>)}
    </dl>
  </>
}

const UUID_PATTERN = "[0-9a-fA-F-]{36}"
const GUIDELINE_LINK = new RegExp(`^/public/guidelines/(${UUID_PATTERN})$`)
const REPORT_LINK = new RegExp(`^/situation-reports/(${UUID_PATTERN})$`)

type LinkedTarget =
  | { kind: "guideline"; id: string }
  | { kind: "report"; id: string }
  | { kind: "external"; url: string }

// Internal routes are mobile-app screens, so they have nothing to open here.
function linkedTarget(item: OutbreakResourceRecord): LinkedTarget | undefined {
  const url = item.url || ""
  const guideline = url.match(GUIDELINE_LINK)
  if (item.resource_type === "guideline" && guideline) return { kind: "guideline", id: guideline[1] }
  const report = url.match(REPORT_LINK)
  if (item.resource_type === "situation_report" && report) return { kind: "report", id: report[1] }
  if (item.resource_type === "approved_external_url" && /^https:\/\//i.test(url)) return { kind: "external", url }
  return undefined
}

function LinkedDocumentActions({ target }: { target: LinkedTarget }) {
  if (target.kind === "external") {
    return <Button asChild size="sm" variant="outline">
      <a href={target.url} target="_blank" rel="noopener noreferrer"><ExternalLink className="mr-2 h-4 w-4" />Open link</a>
    </Button>
  }
  const href = target.kind === "report" ? `/situation-reports/${target.id}` : `/guidelines/${target.id}`
  return <Button asChild size="sm" variant="outline">
    <Link href={href} target="_blank"><Eye className="mr-2 h-4 w-4" />{target.kind === "report" ? "View report" : "View document"}</Link>
  </Button>
}

function errorMessage(value: unknown) {
  const message = value instanceof Error ? value.message : "The action failed."
  return /conflict|modified|lock/i.test(message) ? "Another editor changed this item. Reload before retrying." : message
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : undefined
}
