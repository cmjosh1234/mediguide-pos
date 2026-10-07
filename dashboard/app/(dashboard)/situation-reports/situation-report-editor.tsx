"use client"

import * as React from "react"
import Link from "next/link"
import { CheckCircle2, ExternalLink, Loader2, Save, ShieldCheck } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { apiUrl, getCurrentUser } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import { withDashboardBasePath } from "@/lib/dashboard-path"
import { healthFacilitiesService } from "@/services/health-facilities.service"
import { outbreaksService, type OutbreakRecord } from "@/services/outbreaks.service"
import { situationReportsService, type SituationReportAttachmentRecord, type SituationReportRecord } from "@/services/situation-reports.service"
import { AuditHistory } from "../outbreaks/audit-history"
import { documentKindService, type DocumentKind } from "@/services/document-kinds.service"
import type { DistrictsResponse, RegionsResponse } from "@/types/backend-types"
import { CampaignDraftBuilder } from "../outbreaks/campaign-draft-builder"
import { CORRECTION_STEPS, errorFields, Field, SelectField, UnsavedChanges, WORKFLOW_STEPS, WorkflowSteps, workflowStage } from "../outbreaks/editor-ui"
import { MetricsCard, stripLocalIds, withLocalIds, type MetricDraft } from "../outbreaks/metrics-card"
import { RequiredMark } from "../outbreaks/required-field"
import { ReportAttachments } from "./report-attachments"

// Form fields' element ids, keyed by the backend's JSON field name.
const fieldId = (key: string) => `report-${key}`

// A report can only be published once its related outbreak is live.
const PUBLIC_OUTBREAK_STATUSES = ["published", "active", "monitoring", "contained", "closed"]

function workflowHint(stage: number, correction: boolean) {
  if (correction) {
    return stage === 0
      ? "Change what needs correcting, save the draft, then submit it for review."
      : "Waiting for a reviewer. Approving replaces the published report with this version. The reviewer must be someone other than its author and the person who submitted it."
  }
  switch (stage) {
    case 0:
      return "Fill in the details, add attachments and save the draft, then submit it for review."
    case 1:
      return "Waiting for a reviewer to approve this report. The reviewer must be someone other than its author and the person who submitted it."
    case 2:
      return "Approved. Anyone allowed to publish reports can now publish it."
    case 3:
      return "Live on the public API. To change it, create a correction."
    default:
      return "Withdrawn. This report is no longer public."
  }
}

function statusText(status?: string) {
  return status === "pending_review" ? "in review" : status || "draft"
}

const emptyForm = { outbreak_id: "", region_id: "", district_id: "", title: "", geographic_area: "", summary: "", source_organization: "", source_url: "", source_reference: "", publication_date: "", effective_at: "", data_as_of: "", last_verified_at: "", standalone_allowed: false, key_highlights: "" }
type ReportForm = typeof emptyForm

function formFromReport(value: SituationReportRecord): ReportForm {
  return { outbreak_id: value.outbreak_id || "", region_id: value.region_id || "", district_id: value.district_id || "", title: value.title || "", geographic_area: value.geographic_area || "", summary: value.summary || "", source_organization: value.source_organization || "", source_url: value.source_url || "", source_reference: value.source_reference || "", publication_date: local(value.publication_date), effective_at: local(value.effective_at), data_as_of: local(value.data_as_of), last_verified_at: local(value.last_verified_at), standalone_allowed: Boolean(value.standalone_allowed), key_highlights: (value.key_highlights || []).join("\n") }
}

export function SituationReportEditor({ id }: { id?: string }) {
  const [item, setItem] = React.useState<SituationReportRecord | null>(null)
  // The published report, when this record is a correction of it.
  const [original, setOriginal] = React.useState<SituationReportRecord | null>(null)
  const [outbreaks, setOutbreaks] = React.useState<OutbreakRecord[]>([])
  const [regions, setRegions] = React.useState<RegionsResponse[]>([])
  const [districts, setDistricts] = React.useState<DistrictsResponse[]>([])
  // Bumped after anything that writes audit events, to reload the history.
  const [auditKey, setAuditKey] = React.useState(0)
  const [metrics, setMetrics] = React.useState<MetricDraft[]>([])
  const [savingMetrics, setSavingMetrics] = React.useState(false)
  const [attachments, setAttachments] = React.useState<SituationReportAttachmentRecord[]>([])
  const [documentKinds, setDocumentKinds] = React.useState<DocumentKind[]>([])
  const [loading, setLoading] = React.useState(Boolean(id))
  const [saving, setSaving] = React.useState(false)
  const [error, setError] = React.useState("")
  const [form, setForm] = React.useState<ReportForm>(emptyForm)
  // The report as last loaded or saved, to tell when there are unsaved changes.
  // Metrics aren't part of it: they are saved as soon as they change.
  const [savedForm, setSavedForm] = React.useState<ReportForm>(emptyForm)
  // Errors the backend returned for specific fields, keyed by element id.
  const [serverErrors, setServerErrors] = React.useState<Record<string, string>>({})

  const loadReferences = React.useCallback(async () => {
    const [outbreakPage, regionRows, kinds] = await Promise.all([outbreaksService.list({ page: 1, per_page: 100, sort: "updated_at", order: "desc" }), healthFacilitiesService.regions(), documentKindService.list().catch(() => [])])
    setDocumentKinds(kinds)
    // Reports link to the live outbreak, never to a correction of it.
    setOutbreaks((outbreakPage.items || []).filter(value => !value.supersedes_id)); setRegions(regionRows)
  }, [])
  const hydrate = React.useCallback(async () => {
    setLoading(true); setError("")
    try {
      await loadReferences()
      if (!id) return
      const [value, attached] = await Promise.all([situationReportsService.get(id), situationReportsService.listAttachments(id)])
      const loaded = formFromReport(value)
      setItem(value); setMetrics(withLocalIds((value.metrics || []) as MetricDraft[])); setAttachments(attached || [])
      setOriginal(value.supersedes_id ? await situationReportsService.get(value.supersedes_id).catch(() => null) : null)
      setForm(loaded); setSavedForm(loaded)
    } catch (value) { setError(value instanceof Error ? value.message : "Unable to load report") }
    finally { setLoading(false) }
  }, [id, loadReferences])

  React.useEffect(() => { void hydrate() }, [hydrate])
  React.useEffect(() => {
    if (!form.region_id) { setDistricts([]); return }
    void healthFacilitiesService.districts(form.region_id).then(setDistricts).catch(() => setDistricts([]))
  }, [form.region_id])

  const immutable = item?.status === "published" || item?.status === "withdrawn"
  // A correction replaces the published report when approved, so it is never
  // published itself and keeps the report's related outbreak.
  const isCorrection = Boolean(item?.supersedes_id)
  function field(name: keyof ReportForm, value: string | boolean) {
    setForm(current => ({ ...current, [name]: value }))
    // Editing a field clears its error, and the same error on the other fields
    // that rule named (such as the other of two dates out of order).
    setServerErrors(current => {
      const message = current[fieldId(name)]
      if (!message) return current
      return Object.fromEntries(Object.entries(current).filter(([, other]) => other !== message))
    })
  }
  const errorFor = (key: string) => serverErrors[fieldId(key)]
  const fieldProps = (key: keyof ReportForm) => ({ id: fieldId(key), error: errorFor(key), disabled: immutable })

  function focusField(elementId: string) {
    // Wait for the field to render before focusing it.
    window.setTimeout(() => {
      const element = document.getElementById(elementId)
      if (!element) return
      element.scrollIntoView({ block: "center", behavior: "smooth" })
      element.focus({ preventScroll: true })
    }, 0)
  }
  // Highlights the fields a failed save or workflow step names, replacing any
  // earlier highlights, and moves to the first one.
  function showServerErrors(value: unknown) {
    const fields = errorFields(value)
    setServerErrors(Object.fromEntries(fields.map(({ field, message }) => [fieldId(field), message])))
    if (fields.length > 0) focusField(fieldId(fields[0].field))
  }

  async function save() {
    setSaving(true)
    const sent = form
    try {
      const input = { outbreak_id: form.outbreak_id || undefined, region_id: form.region_id || undefined, district_id: form.district_id || undefined, title: form.title, geographic_area: form.geographic_area, summary: form.summary, source_organization: form.source_organization, source_url: form.source_url, source_reference: form.source_reference, publication_date: iso(form.publication_date), effective_at: iso(form.effective_at), data_as_of: iso(form.data_as_of), last_verified_at: iso(form.last_verified_at), standalone_allowed: form.standalone_allowed, key_highlights: form.key_highlights.split("\n").map(value => value.trim()).filter(Boolean), ...(item ? { lock_version: item.lock_version } : { metrics: stripLocalIds(metrics) }) }
      const result = item ? await situationReportsService.update(item.id!, input) : await situationReportsService.create(input)
      setServerErrors({})
      showToast.success("Report saved", "The report remains unpublished.")
      if (!item) { window.location.assign(withDashboardBasePath(`/situation-reports/${result.id}`)); return }
      setItem(result); setSavedForm(sent)
      refreshAudit()
    } catch (value) {
      showServerErrors(value)
      showToast.error("Unable to save", message(value))
    }
    finally { setSaving(false) }
  }
  async function transition(action: "submit" | "approve" | "publish" | "withdraw" | "correct") {
    if (!item) return
    const reason = ["withdraw", "correct"].includes(action) ? window.prompt(action === "withdraw" ? "Reason for withdrawing this report" : "Reason for this correction")?.trim() : ""
    if (["withdraw", "correct"].includes(action) && !reason) return
    if (action === "approve" && !window.confirm(isCorrection ? "Apply this correction? The published report will be replaced with this version." : "Approve this report for publication?")) return
    if (action === "publish" && !window.confirm("Publish this verified situation report? Published content is immutable.")) return
    if (action === "withdraw" && !window.confirm("Withdraw this published report? Existing links will stop resolving.")) return
    setSaving(true)
    try {
      const next = action === "correct" ? await situationReportsService.correct(item.id!, { lock_version: item.lock_version!, reason }) : await situationReportsService.transition(item.id!, action, { lock_version: item.lock_version!, reason })
      if (action === "correct") {
        showToast.success("Correction created", "Edit the correction, then take it through review.")
        window.location.assign(withDashboardBasePath(`/situation-reports/${next.id}`))
        return
      }
      // Approving a correction returns the published report it was applied to.
      if (action === "approve" && next.id !== item.id) {
        showToast.success("Correction applied", "The published report now shows the corrected version.")
        window.location.assign(withDashboardBasePath(`/situation-reports/${next.id}`))
        return
      }
      setItem(next); setServerErrors({})
      showToast.success("Workflow updated", `Report is now ${statusText(next.status)}.`)
      refreshAudit()
    } catch (value) {
      showServerErrors(value)
      showToast.error("Workflow failed", message(value))
    }
    finally { setSaving(false) }
  }
  async function discardCorrection() {
    if (!item?.id || !item.supersedes_id) return
    if (!window.confirm("Discard this correction? The published report stays as it is.")) return
    setSaving(true)
    try {
      await situationReportsService.remove(item.id, item.lock_version!)
      showToast.success("Correction discarded", "The published report is unchanged.")
      window.location.assign(withDashboardBasePath(`/situation-reports/${item.supersedes_id}`))
    } catch (value) { showToast.error("Correction not discarded", message(value)) }
    finally { setSaving(false) }
  }
  async function refreshAttachments() { if (!item?.id) return; const attached = await situationReportsService.listAttachments(item.id); refreshAudit(); setAttachments(attached || []) }
  function refreshAudit() { setAuditKey(key => key + 1) }
  // The backend replaces the whole metric set, so adding or removing one sends
  // the current set with that change. A saved report saves it straight away;
  // a new one keeps it until its first save.
  async function persistMetrics(next: MetricDraft[], success: { title: string; message: string }, failureTitle: string) {
    if (!item) { setMetrics(next); return true }
    setSavingMetrics(true)
    try {
      const result = await situationReportsService.updateMetrics(item.id!, { metrics: stripLocalIds(next), lock_version: item.lock_version })
      showToast.success(success.title, success.message)
      setItem(result); setMetrics(withLocalIds((result.metrics || []) as MetricDraft[]))
      refreshAudit()
      return true
    } catch (value) {
      showToast.error(failureTitle, message(value))
      return false
    }
    finally { setSavingMetrics(false) }
  }

  if (loading) return <div className="py-20 text-center"><Loader2 className="mr-2 inline h-5 w-5 animate-spin" />Loading…</div>
  if (error) return <div role="alert" className="rounded-md border border-destructive/40 p-5 text-destructive">{error}<Button className="ml-3" variant="outline" onClick={() => void hydrate()}>Retry</Button></div>

  // Mirrors the backend rules (TransitionReport, validatePublishReport) so the
  // page explains what is still needed before each step.
  const currentUserId = String(getCurrentUser()?.id ?? "")
  const stage = workflowStage(item?.status, item?.approved_at)
  const inReview = item?.status === "pending_review"
  const open = item?.status === "draft" || inReview
  const isAuthor = Boolean(item?.author_id && item.author_id === currentUserId)
  const isSubmitter = Boolean(item?.submitted_by && item.submitted_by === currentUserId)
  const relatedOutbreak = outbreaks.find(value => value.id === item?.outbreak_id)
  const publishBlockedByRole = Boolean(item?.approved_by && item.approved_by === currentUserId && relatedOutbreak?.visual_tone === "critical")
  const outbreakNotLive = Boolean(relatedOutbreak && !PUBLIC_OUTBREAK_STATUSES.includes(relatedOutbreak.status || ""))
  const needsAttachment = Boolean(item) && attachments.length === 0 && !item?.report_asset_id
  // Saved values only: the backend checks the stored report, not the form.
  const missingForPublish = item
    ? ([["Geographic area", item.geographic_area], ["Source organization", item.source_organization], ["Source reference", item.source_reference], ["Publication date", item.publication_date], ["Effective at", item.effective_at], ["Last verified", item.last_verified_at]] as const)
      .filter(([, value]) => !String(value ?? "").trim() || String(value).startsWith("0001-01-01"))
      .map(([label]) => label)
    : []
  const dirty = Boolean(item) && !immutable && (Object.keys(form) as Array<keyof ReportForm>).some(key => form[key] !== savedForm[key])
  const workflowNotices = [
    inReview && !item?.approved_at && isAuthor ? "You created this report, so a different reviewer has to approve it." : null,
    inReview && !item?.approved_at && isSubmitter && !isAuthor ? "You submitted this report for review, so a different reviewer has to approve it." : null,
    inReview && item?.approved_at && publishBlockedByRole ? "Reports on a critical outbreak must be published by someone other than the approver." : null,
    open && missingForPublish.length > 0 ? `Before it can be ${isCorrection ? "applied" : "published"}, fill in and save: ${missingForPublish.join(", ")}.` : null,
    open && needsAttachment ? `Before it can be ${isCorrection ? "applied" : "published"}, add at least one attachment.` : null,
    open && outbreakNotLive ? `The related outbreak isn't published yet (it is ${statusText(relatedOutbreak?.status)}). Publish it before this report.` : null,
  ].filter((value): value is string => Boolean(value))
  // Approving a correction publishes its content, so it needs what publishing needs.
  const correctionBlocked = isCorrection && (missingForPublish.length > 0 || needsAttachment || outbreakNotLive)

  return <div className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <div className="flex items-center gap-2">
          <h1 className="text-2xl font-semibold">{item ? item.title || "Untitled report" : "New situation report"}</h1>
          {item ? <Badge variant="outline">{item.status}</Badge> : null}
          {item?.supersedes_id ? <Badge variant="secondary">Correction</Badge> : null}
        </div>
        {item?.supersedes_id ? <p className="text-sm text-muted-foreground">
          Correction of <Link className="font-medium text-foreground underline" href={`/situation-reports/${item.supersedes_id}`}>{original?.title || "the published report"}</Link>
          {item.correction_reason ? ` — ${item.correction_reason}` : "."}
        </p> : <p className="text-sm text-muted-foreground">Draft, review and publish verified situation reports.</p>}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" asChild><Link href="/situation-reports">Back to list</Link></Button>
      </div>
    </div>

    {item ? <Card>
      <CardHeader className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle>{isCorrection ? "Correction workflow" : "Publication workflow"}</CardTitle>
          {isCorrection ? null : <Button variant="ghost" size="sm" asChild>
            <a href={apiUrl(`/api/public/situation-reports/${item.id}`)} target="_blank" rel="noopener noreferrer"><ExternalLink className="mr-2 h-4 w-4" />Preview public API</a>
          </Button>}
        </div>
        <WorkflowSteps stage={stage} steps={isCorrection ? CORRECTION_STEPS : WORKFLOW_STEPS} />
        <p className="text-sm text-muted-foreground">{workflowHint(stage, isCorrection)}</p>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-2">
        {dirty ? <UnsavedChanges message={`You have unsaved changes. ${isCorrection ? "Submitting and approving use" : "Submitting, approving and publishing use"} the saved version, so save the draft first.`} saving={saving} onSave={() => void save()} /> : null}
        <Button variant={item.status === "draft" ? "default" : "outline"} disabled={saving || dirty || item.status !== "draft"} onClick={() => void transition("submit")}>
          Submit for review
        </Button>
        <Button variant={inReview && !item.approved_at ? "default" : "outline"} disabled={saving || dirty || !inReview || Boolean(item.approved_at) || isAuthor || isSubmitter || correctionBlocked} onClick={() => void transition("approve")}>
          <CheckCircle2 className="mr-2 h-4 w-4" />{isCorrection ? "Approve and apply" : "Approve"}
        </Button>
        {isCorrection ? null : <Button variant={inReview && item.approved_at ? "default" : "outline"} disabled={saving || dirty || !inReview || !item.approved_at || publishBlockedByRole || missingForPublish.length > 0 || needsAttachment || outbreakNotLive} onClick={() => void transition("publish")}>
          Publish
        </Button>}
        <div className="ml-auto flex flex-wrap gap-2">
          {isCorrection ? <Button variant="destructive" disabled={saving} onClick={() => void discardCorrection()}>Discard correction</Button> : <>
            <Button variant="outline" disabled={saving || item.status !== "published" || Boolean(item.open_correction_id)} onClick={() => void transition("correct")}>Create correction</Button>
            <Button variant="destructive" disabled={saving || item.status !== "published"} onClick={() => void transition("withdraw")}>Withdraw</Button>
          </>}
        </div>
        {workflowNotices.map(notice => <p key={notice} className="w-full text-sm text-amber-700 dark:text-amber-400">{notice}</p>)}
      </CardContent>
    </Card> : null}

    {immutable ? <div className="flex gap-3 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200">
      <ShieldCheck className="h-5 w-5 shrink-0" />
      <div>
        <strong>Published reports are immutable.</strong>{" "}
        {item?.status === "withdrawn" ? "Withdrawn reports can't be changed." : item?.open_correction_id ? <>
          A correction of this report is in progress. <Link className="font-medium underline" href={`/situation-reports/${item.open_correction_id}`}>Open the correction</Link>
        </> : "Create a correction to change this report."}
      </div>
    </div> : null}

    <Card>
      <CardHeader>
        <CardTitle>Report metadata</CardTitle>
        <p className="text-sm text-muted-foreground"><RequiredMark /> Required to save a draft. Fields marked &ldquo;needed to publish&rdquo; can be left empty for now.</p>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        <SelectField {...fieldProps("outbreak_id")} disabled={immutable || isCorrection} label="Related outbreak" value={form.outbreak_id} onChange={value => field("outbreak_id", value)} options={outbreaks.map(value => ({ id: value.id!, name: value.title || value.id! }))} empty="Standalone report" />
        <div className="pt-8">
          <label className="flex items-center gap-2 text-sm">
            <input id={fieldId("standalone_allowed")} type="checkbox" checked={form.standalone_allowed} disabled={immutable || isCorrection} aria-invalid={Boolean(errorFor("standalone_allowed"))} onChange={event => field("standalone_allowed", event.target.checked)} />
            Explicitly allow standalone report
          </label>
          {errorFor("standalone_allowed") ? <p className="mt-1 text-xs text-destructive">{errorFor("standalone_allowed")}</p> : null}
          {isCorrection ? <p className="mt-1 text-xs text-muted-foreground">A correction keeps the report&apos;s related outbreak.</p> : null}
        </div>
        <Field {...fieldProps("title")} label="Title" required value={form.title} onChange={value => field("title", value)} />
        <Field {...fieldProps("geographic_area")} label="Geographic area" hint="needed to publish" value={form.geographic_area} onChange={value => field("geographic_area", value)} />
        <SelectField {...fieldProps("region_id")} label="Region" value={form.region_id} onChange={value => { field("region_id", value); field("district_id", "") }} options={regions.map(value => ({ id: value.id, name: value.name }))} empty="Select region" />
        <SelectField {...fieldProps("district_id")} label="District" value={form.district_id} onChange={value => field("district_id", value)} options={districts.map(value => ({ id: value.id, name: value.name }))} empty="Select district" />
        <Field {...fieldProps("source_organization")} label="Source organization" hint="needed to publish" value={form.source_organization} onChange={value => field("source_organization", value)} />
        <Field {...fieldProps("source_reference")} label="Source reference" hint="needed to publish" value={form.source_reference} onChange={value => field("source_reference", value)} />
        <Field {...fieldProps("source_url")} label="Source HTTPS URL" hint="approved domains only, e.g. who.int, health.go.ug" value={form.source_url} onChange={value => field("source_url", value)} />
        <Field {...fieldProps("publication_date")} label="Publication date" hint="needed to publish" type="datetime-local" value={form.publication_date} onChange={value => field("publication_date", value)} />
        <Field {...fieldProps("effective_at")} label="Effective at" hint="needed to publish" type="datetime-local" value={form.effective_at} onChange={value => field("effective_at", value)} />
        <Field {...fieldProps("data_as_of")} label="Data as of" type="datetime-local" value={form.data_as_of} onChange={value => field("data_as_of", value)} />
        <Field {...fieldProps("last_verified_at")} label="Last verified" hint="needed to publish" type="datetime-local" value={form.last_verified_at} onChange={value => field("last_verified_at", value)} />
        <TextAreaField id={fieldId("summary")} label="Summary" rows={4} value={form.summary} error={errorFor("summary")} disabled={immutable} onChange={value => field("summary", value)} />
        <TextAreaField id={fieldId("key_highlights")} label="Highlights (one bounded statement per line)" rows={6} value={form.key_highlights} error={errorFor("key_highlights")} disabled={immutable} onChange={value => field("key_highlights", value)} />
        {dirty ? <UnsavedChanges className="md:col-span-2" message="You have unsaved changes." saving={saving} onSave={() => void save()} /> : null}
      </CardContent>
    </Card>

    <MetricsCard
      metrics={metrics}
      saving={savingMetrics}
      onPersist={persistMetrics}
      locked={immutable}
      description={immutable
        ? "Metrics are locked once the report is published. Create a correction to change them."
        : item
          ? "Metrics are saved as soon as you add or remove them, and are reviewed and published with the report. To change a metric, remove it and add it again."
          : "Metrics are saved with the report."}
      dialogDescription={item ? "The metric is saved as soon as you add it." : "The metric is saved with the report."}
      saveLabel={item ? "Save metric" : "Add metric"}
      confirmRemoval={Boolean(item)}
    />

    {/* A saved report offers Save draft beside its unsaved changes; a new one is saved from the end of the form. */}
    {item ? null : <div className="flex justify-end"><Button disabled={saving} onClick={() => void save()}><Save className="mr-2 h-4 w-4" />Save draft</Button></div>}

    {item ? <>
      <ReportAttachments reportId={item.id!} attachments={attachments} documentKinds={documentKinds} locked={immutable} hasUploadedPdf={Boolean(item.report_asset_id)} onChanged={refreshAttachments} />
      <AuditHistory entity="situation_report" id={item.id!} refreshKey={auditKey} />
    </> : null}
    {item?.status === "published" ? <CampaignDraftBuilder contentKey={`situation-report:${item.id}:${item.lock_version}`} kinds={[{ value: "publication", label: "Situation-report publication" }]} create={input => situationReportsService.createCampaign(item.id!, input)} /> : null}
  </div>
}

function TextAreaField({ id, label, rows, value, error, disabled, onChange }: { id: string; label: string; rows: number; value: string; error?: string; disabled?: boolean; onChange: (value: string) => void }) {
  return <div className="md:col-span-2">
    <Label htmlFor={id}>{label}</Label>
    <Textarea id={id} className="mt-2" rows={rows} value={value} disabled={disabled} aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined} onChange={event => onChange(event.target.value)} />
    {error ? <p id={`${id}-error`} className="mt-1 text-xs text-destructive">{error}</p> : null}
  </div>
}
function iso(value: string) { return value ? new Date(value).toISOString() : undefined }
// Go sends an unset date as year 1; the form shows it as empty.
function local(value?: string) { if (!value || value.startsWith("0001-01-01")) return ""; const date = new Date(value); return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16) }
function message(value: unknown) { const text = value instanceof Error ? value.message : "Operation failed"; return /conflict|modified|lock/i.test(text) ? "Another editor changed this report. Reload and review their changes." : text }
