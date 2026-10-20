"use client"

import * as React from "react"
import { toast } from "sonner"
import { AlertTriangle, CheckCircle2, ChevronDown, Download, Play, RotateCcw, Save, Send, ShieldCheck, Upload, XCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { clinicalToolService, type ClinicalToolDefinition, type ClinicalToolVersion, type ValidationIssue } from "@/services/clinical-tool.service"
import { BackendRequestError } from "@/lib/backend-client"
import { WorkflowSteps } from "@/components/ui/workflow-steps"
import { DefinitionOutline } from "./definition-outline"
import { NativeClinicalTool } from "./native-clinical-tool"

const workflowSteps = [
  { status: "draft", label: "Draft" },
  { status: "pending_review", label: "In review" },
  { status: "approved", label: "Approved" },
  { status: "published", label: "Published" },
] as const

function VersionProgress({ status }: { status?: string }) {
  const current = workflowSteps.findIndex((step) => step.status === status)
  if (status && current === -1) {
    return <span className="rounded-md bg-muted px-2 py-1 text-sm capitalize">{status.replaceAll("_", " ")}</span>
  }
  return <WorkflowSteps steps={workflowSteps.map((step) => step.label)} current={current} />
}

function CheckState({ label, passed, saved }: { label: string; passed?: boolean; saved: boolean }) {
  return (
    <span className="flex items-center gap-1.5">
      {passed ? <CheckCircle2 className="h-4 w-4 text-emerald-600" /> : <XCircle className="h-4 w-4 text-muted-foreground" />}
      <span className={passed ? "text-foreground" : "text-muted-foreground"}>
        {label} {passed ? "passed" : saved ? "not passed yet" : "after saving"}
      </span>
    </span>
  )
}

function parseDefinition(raw: string): { definition?: ClinicalToolDefinition; issues: ValidationIssue[] } {
  try {
    const value: unknown = JSON.parse(raw)
    if (!value || typeof value !== "object") return { issues: [{ path: "$", code: "type", message: "Definition must be an object" }] }
    const envelope = value as Record<string, unknown>
    const candidate = envelope.definition ?? value
    if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) return { issues: [{ path: "$", code: "type", message: "Definition must be an object" }] }
    const record = candidate as Record<string, unknown>
    const issues: ValidationIssue[] = []
    for (const key of ["schema_version", "tool_type", "title", "version", "locale"]) {
      if (typeof record[key] !== "string" || !String(record[key]).trim()) issues.push({ path: `$.${key}`, code: "required", message: `${key} is required` })
    }
    for (const key of ["inputs", "sections", "calculation", "rules", "outputs", "interpretations", "test_cases"]) {
      if (!Array.isArray(record[key])) issues.push({ path: `$.${key}`, code: "type", message: `${key} must be an array` })
    }
    if (issues.length) return { issues }
    return { definition: record as ClinicalToolDefinition, issues: [] }
  } catch (error) {
    return { issues: [{ path: "$", code: "json", message: error instanceof Error ? error.message : "Invalid JSON" }] }
  }
}

function isValidationIssueList(value: unknown): value is ValidationIssue[] {
  return Array.isArray(value) && value.every((item) => item && typeof item === "object" && "path" in item && "message" in item)
}

export function ClinicalToolWorkspace({ toolId, toolType, initialVersion, onChanged, versions = [], onSelectVersion, children }: {
  toolId: string
  /** The tool's type; the definition's tool_type must match it. */
  toolType?: string
  initialVersion?: ClinicalToolVersion
  onChanged?: (version: ClinicalToolVersion) => void
  /** Saved versions for the version picker, newest first. */
  versions?: ClinicalToolVersion[]
  onSelectVersion?: (version: ClinicalToolVersion) => void
  /** Rendered under the definition, e.g. review activity. */
  children?: React.ReactNode
}) {
  const [version, setVersion] = React.useState(initialVersion)
  const [raw, setRaw] = React.useState(() => JSON.stringify(initialVersion?.definition ?? {}, null, 2))
  const [summary, setSummary] = React.useState(initialVersion?.change_summary ?? "")
  const [issues, setIssues] = React.useState<ValidationIssue[]>([])
  const [busy, setBusy] = React.useState("")
  const [previewWidth, setPreviewWidth] = React.useState<"mobile" | "tablet" | "desktop">("mobile")
  const [previewDark, setPreviewDark] = React.useState(false)
  const [previewScale, setPreviewScale] = React.useState(1)
  const importRef = React.useRef<HTMLInputElement>(null)
  const parsed = React.useMemo(() => {
    const result = parseDefinition(raw)
    if (result.definition && toolType && result.definition.tool_type !== toolType) {
      const expected = toolType.replaceAll("_", " ")
      return {
        definition: result.definition,
        issues: [{ path: "$.tool_type", code: "type_mismatch", message: `This tool is a ${expected}. Set "tool_type" to "${toolType}", or create a new tool of type ${result.definition.tool_type.replaceAll("_", " ")} for this definition.` }],
      }
    }
    return result
  }, [raw, toolType])

  const updateVersion = (next: ClinicalToolVersion) => { setVersion(next); setRaw(JSON.stringify(next.definition, null, 2)); onChanged?.(next) }
  const action = async (name: string, work: () => Promise<void>) => {
    setBusy(name)
    try { await work() } catch (error) { toast.error(error instanceof Error ? error.message : `${name} failed`) } finally { setBusy("") }
  }
  const save = () => action("save", async () => {
    if (!parsed.definition || parsed.issues.length > 0) throw new Error("Fix the validation issues listed under the editor")
    let next: ClinicalToolVersion
    try {
      next = version
        ? await clinicalToolService.update(version.id, parsed.definition, summary, version.lock_version)
        : await clinicalToolService.create(toolId, parsed.definition, summary)
    } catch (error) {
      // The backend returns the exact problems; list them rather than only a toast.
      if (error instanceof BackendRequestError && isValidationIssueList(error.meta) && error.meta.length > 0) {
        setIssues(error.meta)
        throw new Error("The definition was not saved: see the validation issues under the editor")
      }
      throw error
    }
    updateVersion(next); setIssues([]); toast.success("Draft saved")
  })
  const validate = () => action("validate", async () => {
    if (!version) throw new Error("Save the draft first")
    const result = await clinicalToolService.validate(version.id, version.lock_version)
    setIssues(result.errors ?? []); setVersion({ ...version, validation_passed: result.valid, tests_passed: false, lock_version: result.lock_version })
    if (result.valid) toast.success("Definition is valid"); else toast.error("Definition has validation errors")
  })
  const test = () => action("test", async () => {
    if (!version) throw new Error("Save the draft first")
    const result = await clinicalToolService.test(version.id, version.lock_version)
    setVersion({ ...version, validation_passed: true, tests_passed: result.report.passed, lock_version: result.lock_version })
    if (result.report.passed) toast.success("All saved fixtures passed"); else toast.error("One or more fixtures failed")
  })
  const transition = (name: "submit" | "approve" | "publish" | "withdraw") => action(name, async () => {
    if (!version) throw new Error("Save the draft first")
    if (!window.confirm(`Confirm ${name} for version ${version.semantic_version}?`)) return
    updateVersion(await clinicalToolService.transition(version.id, name, version.lock_version)); toast.success(`Version ${name} complete`)
  })
  const rollbackToLegacy = () => action("legacy rollback", async () => {
    if (!window.confirm("Switch this tool back to its legacy HTML runtime? The published schema version will be retained as an immutable superseded version.")) return
    await clinicalToolService.rollbackToLegacy(toolId)
    toast.success("Legacy runtime restored")
  })

  const exportDefinition = () => {
    const blob = new Blob([raw], { type: "application/json" }); const url = URL.createObjectURL(blob)
    const link = document.createElement("a"); link.href = url; link.download = `${parsed.definition?.title ?? "clinical-tool"}-${parsed.definition?.version ?? "draft"}.json`; link.click(); URL.revokeObjectURL(url)
  }
  const importDefinition = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]; if (!file) return
    const content = await file.text(); const candidate = parseDefinition(content)
    if (!candidate.definition) { setIssues(candidate.issues); toast.error("Imported definition is invalid"); return }
    setRaw(JSON.stringify(candidate.definition, null, 2)); setIssues([]); toast.success("Definition imported; save to persist it")
  }

  const status = version?.status
  const readyToSubmit = Boolean(version?.validation_passed && version?.tests_passed)
  const allIssues = [...parsed.issues, ...issues]

  return <div className="space-y-6">
    <Card>
      <CardContent className="space-y-4 pt-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-4">
            {versions.length > 0 ? (
              <select
                aria-label="Version"
                className="rounded-md border bg-background px-3 py-2 text-sm font-medium"
                value={version?.id}
                onChange={(event) => {
                  const next = versions.find((item) => item.id === event.target.value)
                  if (next) onSelectVersion?.(next)
                }}
              >
                {versions.map((item) => <option key={item.id} value={item.id}>v{item.semantic_version} · {item.status.replaceAll("_", " ")}</option>)}
              </select>
            ) : (
              <span className="text-sm font-medium">No saved version yet</span>
            )}
            <VersionProgress status={status} />
          </div>
          {status === "draft" && <Button onClick={() => transition("submit")} disabled={!readyToSubmit || Boolean(busy)}><Send className="mr-2 h-4 w-4" />Submit for review</Button>}
          {status === "pending_review" && <Button onClick={() => transition("approve")} disabled={Boolean(busy)}><ShieldCheck className="mr-2 h-4 w-4" />Approve</Button>}
          {status === "approved" && <Button onClick={() => transition("publish")} disabled={Boolean(busy)}><Upload className="mr-2 h-4 w-4" />Publish</Button>}
          {status === "published" && <span className="flex items-center gap-1.5 text-sm text-emerald-700 dark:text-emerald-400"><CheckCircle2 className="h-4 w-4" />Live in the app</span>}
        </div>
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 border-t pt-4 text-sm">
          <CheckState label="Validation" passed={version?.validation_passed} saved={Boolean(version)} />
          <CheckState label="Tests" passed={version?.tests_passed} saved={Boolean(version)} />
          <span className="text-muted-foreground">
            {!version
              ? "Write or import a definition, then save a draft."
              : status === "draft" && !readyToSubmit
                ? "Validate and run tests to submit for review."
                : null}
          </span>
        </div>
      </CardContent>
    </Card>

    <div className="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_minmax(340px,.7fr)]">
      <div className="space-y-6">
        <Card>
          <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-3 space-y-0">
            <div className="space-y-1.5">
              <CardTitle>Definition</CardTitle>
              <CardDescription>Clinical Tool Schema v1, written as JSON.</CardDescription>
            </div>
            <div className="flex gap-2">
              <input ref={importRef} className="hidden" type="file" accept="application/json,.json" onChange={importDefinition} />
              <Button size="sm" variant="outline" onClick={() => importRef.current?.click()} disabled={Boolean(busy)}><Upload className="mr-2 h-4 w-4" />Import</Button>
              <Button size="sm" variant="outline" onClick={exportDefinition}><Download className="mr-2 h-4 w-4" />Export</Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <Tabs defaultValue="editor">
              <TabsList>
                <TabsTrigger value="editor">Editor</TabsTrigger>
                <TabsTrigger value="outline">Outline</TabsTrigger>
              </TabsList>
              <TabsContent value="editor" className="mt-4">
                <Textarea aria-label="Clinical tool JSON definition" spellCheck={false} className="min-h-[520px] font-mono text-xs" value={raw} onChange={(event) => setRaw(event.target.value)} />
              </TabsContent>
              <TabsContent value="outline" className="mt-4">
                <DefinitionOutline definition={parsed.definition} />
              </TabsContent>
            </Tabs>
            {allIssues.length > 0 && (
              <div role="alert" className="rounded-md border border-destructive/50 bg-destructive/5 p-3">
                <p className="mb-2 flex items-center gap-2 text-sm font-medium text-destructive"><AlertTriangle className="h-4 w-4" />Validation issues</p>
                <ul className="space-y-1">{allIssues.map((item, index) => <li key={`${item.path}-${index}`} className="text-sm"><code>{item.path}</code>: {item.message}</li>)}</ul>
              </div>
            )}
            <div className="flex flex-wrap items-end gap-3 border-t pt-4">
              <div className="min-w-56 flex-1 space-y-2"><Label htmlFor="change-summary">Change summary</Label><Input id="change-summary" placeholder="What changed in this version?" value={summary} onChange={(event) => setSummary(event.target.value)} /></div>
              <Button variant="outline" onClick={validate} disabled={!version || Boolean(busy)}><ShieldCheck className="mr-2 h-4 w-4" />Validate</Button>
              <Button variant="outline" onClick={test} disabled={!version || Boolean(busy)}><Play className="mr-2 h-4 w-4" />Run tests</Button>
              <Button onClick={save} disabled={Boolean(busy)}><Save className="mr-2 h-4 w-4" />Save draft</Button>
            </div>
          </CardContent>
        </Card>

        {children}

        <details className="group rounded-lg border bg-card">
          <summary className="flex cursor-pointer list-none items-center justify-between p-4 text-sm font-medium">
            More actions
            <ChevronDown className="h-4 w-4 transition-transform group-open:rotate-180" />
          </summary>
          <div className="flex flex-wrap items-start gap-6 border-t p-4 text-sm">
            <div className="max-w-xs space-y-2">
              <Button variant="destructive" size="sm" onClick={() => transition("withdraw")} disabled={!version || version.status !== "superseded" || Boolean(busy)}>Withdraw version</Button>
              <p className="text-xs text-muted-foreground">Only for superseded versions that must no longer be used.</p>
            </div>
            <div className="max-w-xs space-y-2">
              <Button variant="outline" size="sm" onClick={rollbackToLegacy} disabled={Boolean(busy)}><RotateCcw className="mr-2 h-4 w-4" />Restore legacy runtime</Button>
              <p className="text-xs text-muted-foreground">Switches the tool back to its old HTML version.</p>
            </div>
          </div>
        </details>
      </div>

      <Card className="h-fit xl:sticky xl:top-6"><CardHeader><CardTitle>Preview</CardTitle><CardDescription>How the definition runs, using the same restricted evaluator as the app.</CardDescription></CardHeader><CardContent className="space-y-5">
        <div className="flex flex-wrap items-center gap-2" aria-label="Preview settings"><select aria-label="Preview width" className="rounded-md border bg-background px-2 py-2 text-sm" value={previewWidth} onChange={(event) => setPreviewWidth(event.target.value as typeof previewWidth)}><option value="mobile">Mobile</option><option value="tablet">Tablet</option><option value="desktop">Desktop</option></select><Button size="sm" variant="outline" onClick={() => setPreviewDark((value) => !value)}>{previewDark ? "Light theme" : "Dark theme"}</Button><Label className="flex items-center gap-2 text-xs">Text scale<input aria-label="Preview text scale" type="range" min="1" max="2" step="0.25" value={previewScale} onChange={(event) => setPreviewScale(Number(event.target.value))} /></Label></div>
        <div className={`mx-auto space-y-5 rounded-lg border p-4 transition-all ${previewDark ? "dark bg-slate-950 text-slate-50" : "bg-background"} ${previewWidth === "mobile" ? "max-w-sm" : previewWidth === "tablet" ? "max-w-2xl" : "max-w-none"}`} style={{ fontSize: `${previewScale}rem` }}>
        <div className="flex items-center gap-2 text-sm">{version?.validation_passed ? <CheckCircle2 className="h-4 w-4 text-emerald-600" /> : <AlertTriangle className="h-4 w-4 text-amber-600" />}<span>{version?.status ?? "unsaved"} · {parsed.definition?.version ?? version?.semantic_version ?? "no version"}</span></div>
        {parsed.definition ? <NativeClinicalTool definition={parsed.definition} /> : <p className="py-8 text-center text-sm text-muted-foreground">Nothing to preview yet.</p>}
        <p className="text-xs text-muted-foreground">The backend remains the authority for validation, tests and publication.</p>
        </div>
      </CardContent></Card>
    </div>
  </div>
}
