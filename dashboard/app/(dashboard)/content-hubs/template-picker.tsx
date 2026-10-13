"use client"

import * as React from "react"
import { Loader2, Replace } from "lucide-react"

import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Combobox } from "@/components/ui/combobox"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { BackendRequestError } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import { contentHubService, type ContentHubWorkspace, type ContentPillar, type HubTemplate } from "@/services/content-hubs.service"

import { actionable } from "./hub-errors"

/**
 * Chooses a template for the hub. An empty draft hub gets the template's
 * pillars copied in; a draft hub with pillars can replace them after a
 * confirmation with a reason. Published and archived hubs keep their pillars.
 */
export function TemplatePicker({ workspace, templates, onApplied }: { workspace: ContentHubWorkspace; templates: HubTemplate[]; onApplied: () => Promise<void> | void }) {
  const [templateId, setTemplateId] = React.useState("")
  const [applying, setApplying] = React.useState(false)
  const [replacing, setReplacing] = React.useState(false)
  const draft = workspace.hub.status === "draft"
  const hasPillars = workspace.pillars.length > 0
  const template = templates.find(value => value.id === templateId)

  async function apply() {
    if (!templateId) return
    setApplying(true)
    try {
      await contentHubService.applyTemplate(workspace.hub.id, templateId, workspace.hub.lock_version)
      showToast.success("Template applied", "The copied pillars can now be renamed, nested and reordered.")
      setTemplateId("")
      await onApplied()
    } catch (error) {
      showToast.error("Template not applied", actionable(error))
    } finally { setApplying(false) }
  }

  const hint = !draft
    ? `Templates can only be applied to a draft hub. This hub is ${workspace.hub.status === "active" ? "published" : workspace.hub.status}, so change its pillars below instead.`
    : hasPillars
      ? "This hub already has pillars. Replacing the template deletes them and their resource assignments first."
      : "Copies the template's pillars into this hub. You can rename, nest, reorder or delete them afterwards."
  return <div className="space-y-2">
    <div className="flex gap-2">
      <div className="flex-1">
        <Combobox
          options={templates.map(value => ({ value: value.id, label: value.name }))}
          value={templateId}
          onValueChange={setTemplateId}
          placeholder="Choose a template"
          searchPlaceholder="Search template..."
          emptyText="No template found."
          clearable
          disabled={!draft}
        />
      </div>
      {draft && hasPillars ? (
        <Button variant="outline" disabled={!templateId} onClick={() => setReplacing(true)}>
          <Replace className="mr-2 h-4 w-4" />Replace template
        </Button>
      ) : (
        <Button variant="outline" disabled={!templateId || !draft || applying} onClick={() => void apply()}>
          {applying ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}Apply template
        </Button>
      )}
    </div>
    <p className="text-sm text-muted-foreground">{hint}</p>
    <ReplaceTemplateDialog
      hubId={workspace.hub.id}
      lockVersion={workspace.hub.lock_version}
      template={replacing ? template || null : null}
      pillars={workspace.pillars}
      onOpenChange={open => { if (!open) setReplacing(false) }}
      onReplaced={async () => { setTemplateId(""); await onApplied() }}
    />
  </div>
}

/** Confirms replacing a draft hub's pillars with a template's, with a reason. The assigned resources themselves are kept. */
export function ReplaceTemplateDialog({ hubId, lockVersion, template, pillars, onOpenChange, onReplaced }: { hubId: string; lockVersion: number; template: HubTemplate | null; pillars: ContentPillar[]; onOpenChange: (open: boolean) => void; onReplaced: () => Promise<void> | void }) {
  const [reason, setReason] = React.useState("")
  const [replacing, setReplacing] = React.useState(false)
  const [error, setError] = React.useState("")

  React.useEffect(() => { setReason(""); setError("") }, [template?.id])

  async function replace() {
    if (!template) return
    setReplacing(true); setError("")
    try {
      await contentHubService.applyTemplate(hubId, template.id, lockVersion, reason.trim())
      showToast.success("Template replaced", "The previous pillars and your reason are recorded in the audit history.")
      onOpenChange(false)
      await onReplaced()
    } catch (value) {
      setError(value instanceof BackendRequestError && value.status === 409 ? "Someone changed this hub after the page loaded. Close this, reload the page and try again." : value instanceof Error ? value.message : "Unable to replace the template")
    } finally { setReplacing(false) }
  }

  const assignments = pillars.reduce((total, row) => total + (row.items?.length || 0), 0)
  const removed = [count(pillars.length, "pillar"), count(assignments, "resource assignment")].filter(Boolean)
  const added = template?.pillars || []
  return <AlertDialog open={template !== null} onOpenChange={open => { if (!replacing) onOpenChange(open) }}>
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Replace this hub&apos;s pillars with “{template?.name}”?</AlertDialogTitle>
        <AlertDialogDescription asChild>
          <div className="space-y-2 text-sm">
            <p><span className="font-medium text-foreground">Deleted:</span> {removed.join(" and ")} ({pillars.map(row => row.name).join(", ")}).</p>
            <p><span className="font-medium text-foreground">Added:</span> {count(added.length, "pillar")} from the template{added.length ? ` (${added.map(row => row.name).join(", ")})` : ""}.</p>
            <p><span className="font-medium text-foreground">Kept:</span> the guidelines and other resources that were assigned. Only their place in this hub is removed, so assign them again afterwards.</p>
            <p>This can&apos;t be undone from the dashboard.</p>
          </div>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <div className="space-y-2">
        <Label htmlFor="replace-template-reason">Reason</Label>
        <Textarea id="replace-template-reason" rows={3} value={reason} onChange={event => setReason(event.target.value)} placeholder="For example: this hub should follow the surveillance layout" disabled={replacing} />
      </div>
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      <AlertDialogFooter>
        <AlertDialogCancel disabled={replacing}>Cancel</AlertDialogCancel>
        <Button variant="destructive" disabled={replacing || !reason.trim()} onClick={() => void replace()}>
          {replacing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Replace className="mr-2 h-4 w-4" />}Replace template
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
}

function count(value: number, noun: string) {
  return value ? `${value} ${noun}${value === 1 ? "" : "s"}` : ""
}
