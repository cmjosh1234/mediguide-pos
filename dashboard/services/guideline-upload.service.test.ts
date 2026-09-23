import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { webcrypto } from "node:crypto"
const request = vi.fn()
vi.mock("@/lib/backend-client", () => ({ getBackendClient: () => ({ request }) }))
import { uploadGuidelineDirect } from "./guideline-upload.service"

const file = { name: "guide.md", size: 4, arrayBuffer: async () => new TextEncoder().encode("test").buffer, slice: () => new Blob(["test"]) } as File
const session = { id: "session", filename: "guide.md", status: "uploading", checksum: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", size_bytes: 4, part_size: 8 << 20, expires_at: "2099-01-01" }
describe("resumable guideline uploads", () => {
  beforeEach(() => { request.mockReset(); vi.stubGlobal("crypto", webcrypto) })
  afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })
  it("refreshes an expired signed URL without restarting the session", async () => {
    const urls: string[] = []
    class FakeXHR {
      status = 0
      upload = { onprogress: null }
      onload?: () => void
      open(_method: string, url: string) { urls.push(url) }
      send() { this.status = urls.length === 1 ? 403 : 200; queueMicrotask(() => this.onload?.()) }
      abort() {}
    }
    vi.stubGlobal("XMLHttpRequest", FakeXHR)
    request.mockResolvedValueOnce([session])
      .mockResolvedValueOnce({ ...session, parts: [] })
      .mockResolvedValueOnce({ url: "https://storage.test/expired" })
      .mockResolvedValueOnce({ url: "https://storage.test/fresh" })
      .mockResolvedValueOnce({ id: "one-job" })
    expect(await uploadGuidelineDirect("version", file)).toEqual({ id: "one-job" })
    expect(urls).toEqual(["https://storage.test/expired", "https://storage.test/fresh"])
    expect(request.mock.calls.filter(([url]) => url.endsWith("/complete"))).toHaveLength(1)
  })
  it("resumes after a connection interruption without resending stored bytes", async () => {
    const sends: number[] = []
    class FakeXHR {
      status = 200
      upload = { onprogress: null }
      onload?: () => void
      open() {}
      send(blob: Blob) { sends.push(blob.size); queueMicrotask(() => this.onload?.()) }
      abort() {}
    }
    vi.stubGlobal("XMLHttpRequest", FakeXHR)
    request.mockResolvedValueOnce([session])
      .mockResolvedValueOnce({ ...session, part_size: 2, parts: [{ number: 1, size: 2, etag: "stored" }] })
      .mockResolvedValueOnce({ url: "https://storage.test/part-2" })
      .mockResolvedValueOnce({ id: "one-job" })
    const resumable = { ...file, slice: (start: number, end: number) => new Blob(["test".slice(start, end)]) } as File
    await uploadGuidelineDirect("version", resumable)
    expect(sends).toEqual([2])
    expect(request.mock.calls.some(([url]) => url.endsWith("/parts/1"))).toBe(false)
  })
  it("does not resend completed parts", async () => {
    request.mockResolvedValueOnce([session]).mockResolvedValueOnce({ ...session, parts: [{ number: 1, size: 4 }] }).mockResolvedValueOnce({ id: "job" })
    expect(await uploadGuidelineDirect("version", file)).toEqual({ id: "job" })
    expect(request).toHaveBeenLastCalledWith("/api/v2/guideline-versions/version/uploads/session/complete", expect.objectContaining({ method: "POST" }))
    expect(request).toHaveBeenCalledTimes(3)
  })
  it("recovers a completed storage object without trying to sign parts", async () => {
    request.mockResolvedValueOnce([session]).mockResolvedValueOnce({ ...session, object_complete: true }).mockResolvedValueOnce({ id: "job" })
    await uploadGuidelineDirect("version", file)
    expect(request.mock.calls.some(([url]) => url.includes("/parts/"))).toBe(false)
  })
  it("reconciles a lost completion response to the same job", async () => {
    request.mockResolvedValueOnce([]).mockResolvedValueOnce({ ...session, status: "completed" }).mockResolvedValueOnce({ ...session, status: "completed", job: { id: "same-job" } })
    expect(await uploadGuidelineDirect("version", file, { sessionId: "session" })).toEqual({ id: "same-job" })
    expect(request.mock.calls.some(([, init]) => init?.method === "POST")).toBe(false)
  })
  it("rejects a different file when explicitly resuming", async () => {
    request.mockResolvedValueOnce([]).mockResolvedValueOnce({ ...session, checksum: "other" })
    await expect(uploadGuidelineDirect("version", file, { sessionId: "session" })).rejects.toThrow("original file")
  })
  it("stops before creating a session when paused during hashing", async () => {
    const abort = new AbortController(); abort.abort()
    await expect(uploadGuidelineDirect("version", file, { signal: abort.signal })).rejects.toMatchObject({ name: "AbortError" })
    expect(request).not.toHaveBeenCalled()
  })
})
