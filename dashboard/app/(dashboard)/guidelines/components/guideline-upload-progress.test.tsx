import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { GuidelineUploadProgress } from "./guideline-upload-progress"
const request = vi.fn()
vi.mock("@/lib/backend-client", () => ({ getBackendClient: () => ({ request, authStore: { record: { id: "editor" } } }) }))
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }))

describe("guideline upload progress", () => {
  afterEach(cleanup)
  beforeEach(() => { request.mockReset(); localStorage.clear() })

  it("restores a paused session after refresh and permits explicit cancellation", async () => {
    request.mockResolvedValueOnce([{ id: "session", filename: "guide.pdf", status: "uploading", expires_at: "2099-01-01" }]).mockResolvedValueOnce({ id: "session", parts: [] }).mockResolvedValueOnce({ aborted: true })
    render(<GuidelineUploadProgress versionId="version" file={null} submitting={false} onSubmit={vi.fn()} />)
    expect(await screen.findByText(/Resume guide.pdf/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Resume upload" })).toBeDisabled()
    fireEvent.click(screen.getByRole("button", { name: "Cancel upload" }))
    expect(await screen.findByText("Upload canceled.")).toBeInTheDocument()
    expect(request).toHaveBeenLastCalledWith("/api/v2/guideline-versions/version/uploads/session", { method: "DELETE" })
  })

  it("separates queued processing from transfer and persists the returned job", async () => {
    request.mockResolvedValueOnce([])
    const onSubmit = vi.fn().mockResolvedValue({ id: "job", status: "queued", progress_percent: 0 })
    render(<GuidelineUploadProgress versionId="version" file={new File(["test"], "guide.pdf")} submitting={false} onSubmit={onSubmit} />)
    await waitFor(() => expect(screen.getByRole("button", { name: "Upload source" })).toBeEnabled())
    fireEvent.click(screen.getByRole("button", { name: "Upload source" }))
    expect(await screen.findByText("Processing: queued")).toBeInTheDocument()
    expect(JSON.parse(localStorage.getItem("mediguide-upload:editor:version")!)).toMatchObject({ job: "job" })
    expect(screen.getByText(/no content is approved or published automatically/)).toBeInTheDocument()
  })

  it("restores terminal job state without starting another transfer", async () => {
    localStorage.setItem("mediguide-upload:editor:version", JSON.stringify({ job: "job" }))
    request.mockResolvedValueOnce({ id: "job", status: "completed", progress_percent: 100 })
    const onSubmit = vi.fn()
    render(<GuidelineUploadProgress versionId="version" file={null} submitting={false} onSubmit={onSubmit} />)
    expect(await screen.findByText("Ready for editorial review")).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
