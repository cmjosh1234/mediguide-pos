"use client"

import * as React from "react"
import { Button } from "@/components/ui/button"
import { getBackendClient } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import type { IngestionJobRecord } from "@/services/guideline-documents.service"
import { abortSourceUpload, listSourceUploads, sourceUploadStatus, type UploadOptions, type UploadProgress } from "@/services/guideline-upload.service"

const terminal = (status: string) => ["completed", "failed", "canceled", "superseded"].includes(status)

export function GuidelineUploadProgress({ versionId, file, submitting, onSubmit }: {
  versionId: string; file: File | null; submitting: boolean
  onSubmit: (file: File, options?: UploadOptions) => Promise<IngestionJobRecord | void>
}) {
  const [session, setSession] = React.useState<string>()
  const [progress, setProgress] = React.useState<UploadProgress>()
  const [job, setJob] = React.useState<IngestionJobRecord>()
  const [message, setMessage] = React.useState("")
  const [restoring, setRestoring] = React.useState(true)
  const [canceling, setCanceling] = React.useState(false)
  const controller = React.useRef<AbortController | null>(null)
  const activeSession = React.useRef<string | undefined>(undefined)
  const key = `mediguide-upload:${getBackendClient().authStore.record?.id}:${versionId}`
  const jobId = job?.id
  const jobStatus = job?.status
  const remember = (value: { session?: string; job?: string }) => { try { localStorage.setItem(key, JSON.stringify(value)) } catch { /* Storage may be disabled. Server sessions remain resumable. */ } }

  React.useEffect(() => {
    let disposed = false
    async function restore() {
      try {
        let saved: { session?: string; job?: string } = {}
        try { saved = JSON.parse(localStorage.getItem(key) || "{}") } catch { /* Ignore stale browser state. */ }
        if (saved.job) {
          const existing = await getBackendClient().request<IngestionJobRecord>(`/api/v2/guideline-versions/${versionId}/upload-jobs/${saved.job}`)
          if (!disposed) setJob(existing)
          return
        }
        const rows = await listSourceUploads(versionId)
        const row = rows.find(row => row.id === saved.session && row.status !== "aborted") ?? rows.find(row => row.status === "uploading" && Date.parse(row.expires_at) > Date.now())
        if (row && !disposed) {
          activeSession.current = row.id; setSession(row.id)
          const state = await sourceUploadStatus(versionId, row.id)
          if (!disposed) {
            if (state.job) setJob(state.job)
            else setMessage(`Resume ${row.filename}: select the same file. Completed parts are preserved for 24 hours.`)
          }
        }
      } catch { /* Older deployments still support the standard upload form. */ }
      finally { if (!disposed) setRestoring(false) }
    }
    void restore()
    return () => { disposed = true; controller.current?.abort() }
  }, [versionId, key])

  React.useEffect(() => {
    if (!jobId || !jobStatus || terminal(jobStatus)) return
    let disposed = false
    let timer: ReturnType<typeof setTimeout>
    const poll = async () => {
      try {
        const value = await getBackendClient().request<IngestionJobRecord>(`/api/v2/guideline-versions/${versionId}/upload-jobs/${jobId}`)
        if (disposed) return
        setJob(value)
        if (terminal(value.status)) {
          if (value.status === "completed") showToast.success("Document ready for review", "Extraction and indexing finished. Editorial approval is still required.")
          else showToast.error("Document processing stopped", value.error || value.status)
          return
        }
      } catch { if (!disposed) setMessage("Processing continues on the server. Reconnecting to status…") }
      if (!disposed) timer = setTimeout(poll, 3000)
    }
    timer = setTimeout(poll, 1500)
    return () => { disposed = true; clearTimeout(timer) }
  }, [jobId, jobStatus, versionId])

  async function start() {
    if (!file) return
    const abort = new AbortController(); controller.current = abort; setMessage("")
    try {
      const result = await onSubmit(file, {
        signal: abort.signal, sessionId: session,
        onProgress: setProgress,
        onSession: id => { activeSession.current = id; setSession(id); remember({ session: id }) },
      })
      if (result) { setJob(result); setProgress(undefined); remember({ session: activeSession.current, job: result.id }) }
    } catch (error) {
      if (abort.signal.aborted) setMessage("Upload paused. Select the same file to resume completed parts.")
      else { const text = error instanceof Error ? error.message : "Upload failed. Please retry."; setMessage(text); showToast.error("Upload failed", text) }
    } finally { controller.current = null }
  }
  async function cancel() {
    setCanceling(true)
    try {
      if (session) await abortSourceUpload(versionId, session)
      activeSession.current = undefined; setSession(undefined); setProgress(undefined); setMessage("Upload canceled.")
      try { localStorage.removeItem(key) } catch { /* Optional persistence. */ }
    } catch (error) { showToast.error("Cancel failed", error instanceof Error ? error.message : "Refresh the upload status before retrying.") }
    finally { setCanceling(false) }
  }
  if (job) return <div className="space-y-3" role="status" aria-live="polite">
    <p>{job.status === "completed" ? "Ready for editorial review" : `Processing: ${job.progress_stage || job.status}`}</p>
    <progress className="w-full" max={100} value={job.progress_percent || 0} aria-label="Document processing progress" />
    <p className="text-sm text-muted-foreground">{job.error || "You can close this dialog. Processing continues on the server; no content is approved or published automatically."}</p>
    {terminal(job.status) && <Button variant="outline" onClick={() => { setJob(undefined); setSession(undefined); activeSession.current = undefined; remember({}) }}>Start another upload</Button>}
  </div>
  return <div className="space-y-3">
    <div role="status" aria-live="polite">
      {progress && <><p>{progress.stage === "hashing" ? "Checking file…" : progress.stage === "verifying" ? "Verifying stored file and queuing processing…" : `${Math.round(progress.loaded / progress.total * 100)}% uploaded · ${(progress.bytesPerSecond / 1048576).toFixed(1)} MB/s`}</p><progress className="w-full" max={progress.total} value={progress.stage === "hashing" ? undefined : progress.loaded} aria-label="Source upload progress" /></>}
      {submitting && !progress && <p>Uploading source…</p>}
      {message && <p className="text-sm text-muted-foreground">{message}</p>}
    </div>
    <div className="flex flex-wrap gap-2">
      <Button onClick={() => void start()} disabled={!file || submitting || restoring || canceling}>{session ? "Resume upload" : "Upload source"}</Button>
      {submitting && progress?.stage === "uploading" && <Button variant="outline" onClick={() => controller.current?.abort()}>Pause</Button>}
      {session && !submitting && <Button variant="outline" disabled={canceling} onClick={() => void cancel()}>Cancel upload</Button>}
    </div>
  </div>
}
