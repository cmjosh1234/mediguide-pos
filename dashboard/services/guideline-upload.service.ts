import { getBackendClient } from "@/lib/backend-client"
import type { IngestionJobRecord } from "@/services/guideline-documents.service"

export type UploadProgress = { stage: "hashing" | "uploading" | "verifying"; loaded: number; total: number; bytesPerSecond: number }
export type UploadOptions = { signal?: AbortSignal; sessionId?: string; onProgress?: (value: UploadProgress) => void; onSession?: (id: string) => void }
export type SourceUpload = {
  id: string; version_id: string; filename: string; checksum: string; size_bytes: number
  part_size: number; status: string; expires_at: string; job_id?: string
  object_complete?: boolean
  parts?: Array<{ number: number; size: number; etag: string }>
  job?: IngestionJobRecord & { progress_stage?: string; progress_percent?: number; metrics?: Record<string, unknown> }
}
const path = (version: string) => `/api/v2/guideline-versions/${version}/uploads`
export const listSourceUploads = (version: string) => getBackendClient().request<SourceUpload[]>(path(version))
export const sourceUploadStatus = (version: string, id: string) => getBackendClient().request<SourceUpload>(`${path(version)}/${id}`)
export const abortSourceUpload = (version: string, id: string) => getBackendClient().request(`${path(version)}/${id}`, { method: "DELETE" })

function putPart(url: string, part: Blob, signal: AbortSignal | undefined, progress: (bytes: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const abort = () => xhr.abort()
    if (signal?.aborted) { reject(new DOMException("Upload paused", "AbortError")); return }
    xhr.open("PUT", url)
    xhr.timeout = 120_000
    // Signed storage requests intentionally carry no dashboard credentials.
    xhr.upload.onprogress = event => progress(event.loaded)
    const cleanup = () => signal?.removeEventListener("abort", abort)
    xhr.onload = () => { cleanup(); if (xhr.status >= 200 && xhr.status < 300) resolve(); else reject(new Error(`Part upload failed (${xhr.status})`)) }
    xhr.onerror = () => { cleanup(); reject(new Error("Storage could not be reached. Check your connection and resume.")) }
    xhr.ontimeout = () => { cleanup(); reject(new Error("Part upload timed out. Resume to retry.")) }
    xhr.onabort = () => { cleanup(); reject(new DOMException("Upload paused", "AbortError")) }
    signal?.addEventListener("abort", abort, { once: true })
    xhr.send(part)
  })
}

export async function uploadGuidelineDirect(version: string, file: File, options: UploadOptions = {}): Promise<IngestionJobRecord> {
  const api = getBackendClient()
  const { signal, onProgress, onSession } = options
  const checkAbort = () => { if (signal?.aborted) throw new DOMException("Upload paused", "AbortError") }
  onProgress?.({ stage: "hashing", loaded: 0, total: file.size, bytesPerSecond: 0 })
  const checksum = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer())), b => b.toString(16).padStart(2, "0")).join("")
  checkAbort()
  const sessions = await listSourceUploads(version)
  const existing = options.sessionId ? await sourceUploadStatus(version, options.sessionId) : sessions.find(row => row.status === "uploading" && Date.parse(row.expires_at) > Date.now() && row.checksum === checksum && row.size_bytes === file.size)
  if (existing && (existing.checksum !== checksum || existing.size_bytes !== file.size)) throw new Error("Select the original file to resume this upload, or cancel it before choosing another file.")
  let session = existing ?? await api.request<SourceUpload>(path(version), { method: "POST", signal, body: JSON.stringify({ filename: file.name, size_bytes: file.size, checksum }) })
  onSession?.(session.id)
  session = await sourceUploadStatus(version, session.id)
  if (session.job) return session.job
  if (session.status !== "uploading" || Date.parse(session.expires_at) <= Date.now()) throw new Error("This upload expired or was canceled. Start a new upload.")
  if (session.object_complete) {
    onProgress?.({ stage: "verifying", loaded: file.size, total: file.size, bytesPerSecond: 0 })
    return api.request<IngestionJobRecord>(`${path(version)}/${session.id}/complete`, { method: "POST", signal })
  }
  const parts = session.parts ?? []
  const complete = new Set(parts.map(part => part.number))
  let uploaded = parts.reduce((total, part) => total + part.size, 0)
  const resumedBytes = uploaded
  const started = performance.now()
  const active = new Map<number, number>()
  const report = () => {
    const loaded = uploaded + Array.from(active.values()).reduce((sum, bytes) => sum + bytes, 0)
    onProgress?.({ stage: "uploading", loaded, total: file.size, bytesPerSecond: (loaded - resumedBytes) / Math.max(0.1, (performance.now() - started) / 1000) })
  }
  report()
  const pending = Array.from({ length: Math.ceil(file.size / session.part_size) }, (_, index) => index + 1).filter(number => !complete.has(number))
  // Three transfers bound network/memory use while retries only resend one part.
  let failed: unknown
  await Promise.all(Array.from({ length: Math.min(3, pending.length) }, async () => {
    while (pending.length && !failed) {
      const number = pending.shift()!
      const part = file.slice((number - 1) * session.part_size, number * session.part_size)
      try {
        for (let attempt = 0; ; attempt++) {
          checkAbort()
          try {
            const { url } = await api.request<{ url: string }>(`${path(version)}/${session.id}/parts/${number}`, { method: "POST", signal })
            await putPart(url, part, signal, bytes => { active.set(number, bytes); report() })
            break
          } catch (error) {
            active.delete(number); report()
            if (signal?.aborted || attempt >= 2) throw error
            await new Promise(resolve => setTimeout(resolve, 500 * 2 ** attempt))
          }
        }
        uploaded += part.size; active.delete(number); report()
      } catch (error) { failed = error }
    }
  }))
  if (failed) throw failed
  checkAbort()
  onProgress?.({ stage: "verifying", loaded: file.size, total: file.size, bytesPerSecond: 0 })
  return api.request<IngestionJobRecord>(`${path(version)}/${session.id}/complete`, { method: "POST", signal })
}
