"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { PageHeader } from "@/components/ui/page-header"
import { ClinicalToolWorkspace } from "@/components/clinical-tools/clinical-tool-workspace"
import { clinicalToolService, type ClinicalToolAuditEntry, type ClinicalToolVersion } from "@/services/clinical-tool.service"
import { ClinicalToolAuditEntryItem } from "@/components/clinical-tools/audit-entry"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { EmptyState } from "@/components/ui/empty-state"
import { toast } from "sonner"
import { RiskLevelNotice } from "@/components/clinical-tools/risk-level"
import { useDomainRecord } from "@/hooks/use-domain-record"
import { calculatorService } from "@/services/calculator.service"

type AuditItem = ClinicalToolAuditEntry

export default function ClinicalToolAuthorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = React.use(params)
  const { record: tool } = useDomainRecord("calculators", id, calculatorService.get)
  const router = useRouter()
  const [version, setVersion] = React.useState<ClinicalToolVersion>()
  const [versions, setVersions] = React.useState<ClinicalToolVersion[]>([])
  const [audit, setAudit] = React.useState<AuditItem[]>([])
  const [auditLoading, setAuditLoading] = React.useState(false)
  const [auditError, setAuditError] = React.useState("")
  const auditRequest = React.useRef(0)
  const [comment, setComment] = React.useState("")
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState("")
  React.useEffect(() => { clinicalToolService.versions(id).then((items) => { setVersions(items); setVersion(items.find((item) => item.status === "draft") ?? items[0]) }).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Unable to load versions")).finally(() => setLoading(false)) }, [id])
  const versionId = version?.id
  const loadAudit = React.useCallback(async () => {
    const request = ++auditRequest.current
    setAudit([])
    setAuditError("")
    if (!versionId) { setAuditLoading(false); return }
    setAuditLoading(true)
    try {
      const items = await clinicalToolService.audit(versionId)
      if (request === auditRequest.current) setAudit(items)
    } catch (reason) {
      if (request === auditRequest.current) setAuditError(reason instanceof Error ? reason.message : "Unable to load review activity")
    } finally {
      if (request === auditRequest.current) setAuditLoading(false)
    }
  }, [versionId])
  React.useEffect(() => { void loadAudit(); return () => { auditRequest.current += 1 } }, [loadAudit])
  const addComment = async () => {
    if (!versionId || !comment.trim()) return
    try { await clinicalToolService.reviewComment(versionId, comment.trim()); setComment(""); await loadAudit(); toast.success("Review comment added") }
    catch (reason) { toast.error(reason instanceof Error ? reason.message : "Unable to add review comment") }
  }
  const changed = (next: ClinicalToolVersion) => { setVersion(next); setVersions((items) => items.some((item) => item.id === next.id) ? items.map((item) => item.id === next.id ? next : item) : [next, ...items]) }
  const published = versions.find((item) => item.status === "published")
  const reviewActivity = (
    <Card>
      <CardHeader>
        <CardTitle>Review activity</CardTitle>
        <CardDescription>Changes and review comments for this version. Comments cannot be edited once added.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {auditLoading ? (
          <p role="status" className="text-sm text-muted-foreground">Loading review activity…</p>
        ) : auditError ? (
          <div role="alert" className="space-y-2 text-sm"><p>{auditError}</p><Button variant="outline" size="sm" onClick={() => void loadAudit()}>Try again</Button></div>
        ) : audit.length === 0 ? (
          <EmptyState title="No review activity yet" description="Saved changes and review comments will appear here." />
        ) : (
          <ul className="max-h-[28rem] space-y-2 overflow-auto">
            {audit.map((item) => <ClinicalToolAuditEntryItem key={item.id} item={item} />)}
          </ul>
        )}
        {version && version.status !== "withdrawn" && (
          <div className="space-y-2 border-t pt-4">
            <Textarea aria-label="Review comment" placeholder="Add a clinical review comment…" maxLength={4000} rows={3} value={comment} onChange={(event) => setComment(event.target.value)} />
            <div className="flex justify-end"><Button size="sm" variant="outline" disabled={!comment.trim()} onClick={addComment}>Add comment</Button></div>
          </div>
        )}
      </CardContent>
    </Card>
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title={tool?.name ?? "Clinical tool"}
        description="Authoring workspace: write, test, review and publish this tool's definition."
        showBackButton
        onBack={() => router.push(`/decision-tools/${id}`)}
      />
      {tool && <RiskLevelNotice level={tool.risk_level} />}
      {loading ? (
        <p className="text-sm text-muted-foreground">Loading version history…</p>
      ) : error ? (
        <p role="alert" className="text-destructive">{error}</p>
      ) : (
        <ClinicalToolWorkspace key={version?.id ?? "new"} toolId={id} toolType={tool?.type} initialVersion={version} onChanged={changed} versions={versions} onSelectVersion={setVersion}>
          {reviewActivity}
          {published && version && published.id !== version.id && (
            <details className="rounded-lg border bg-card">
              <summary className="cursor-pointer p-4 text-sm font-medium">Compare v{version.semantic_version} with the published v{published.semantic_version}</summary>
              <div className="grid gap-4 border-t p-4 lg:grid-cols-2">
                <div className="space-y-2"><p className="text-xs font-medium text-muted-foreground">Published v{published.semantic_version}</p><pre className="max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs">{JSON.stringify(published.definition, null, 2)}</pre></div>
                <div className="space-y-2"><p className="text-xs font-medium text-muted-foreground">This version, v{version.semantic_version}</p><pre className="max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs">{JSON.stringify(version.definition, null, 2)}</pre></div>
              </div>
            </details>
          )}
        </ClinicalToolWorkspace>
      )}
    </div>
  )
}
