import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { GuidelineRevisionAction } from "./guideline-revision-action"
import type { GuidelineDocumentRecord } from "@/services/guideline-documents.service"

const { revise, push, success } = vi.hoisted(() => ({ revise: vi.fn(), push: vi.fn(), success: vi.fn() }))
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }))
vi.mock("@/lib/toast", () => ({ showToast: { success } }))
vi.mock("@/services/guideline-documents.service", async importOriginal => ({
  ...await importOriginal<typeof import("@/services/guideline-documents.service")>(),
  GuidelineDocumentsService: { reviseDocument: revise },
}))
const document = {
  id: "doc", title: "Withdrawn guidance", withdrawn_at: "2026-10-16T12:00:00Z", withdrawn_version_id: "old",
  versions: [{ id: "old", version: "1", status: "published" }],
} as GuidelineDocumentRecord
beforeEach(() => vi.resetAllMocks())
afterEach(cleanup)

it("creates a revision and discloses omitted content before opening the editor", async () => {
  revise.mockResolvedValue({ version: { id: "draft", version: "2", revision_warnings: ["Omitted unreviewed section: References"] }, existing: false })
  const onChanged = vi.fn().mockResolvedValue(undefined)
  render(<GuidelineRevisionAction document={document} onChanged={onChanged} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  expect(screen.getByLabelText("New version number")).toHaveValue("2")
  fireEvent.change(screen.getByLabelText("New version number"), { target: { value: " 2.5 " } })
  fireEvent.click(screen.getByRole("button", { name: "Create revision draft" }))
  await waitFor(() => expect(revise).toHaveBeenCalledWith("doc", "old", "2.5"))
  expect(await screen.findByText("Omitted unreviewed section: References")).toBeInTheDocument()
  expect(push).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole("button", { name: "Open draft" }))
  expect(push).toHaveBeenCalledWith("/guidelines/doc/versions/draft/markdown")
  expect(onChanged).toHaveBeenCalledOnce()
})

it("continues an existing revision without creating another draft", () => {
  render(<GuidelineRevisionAction document={{ ...document, versions: [...document.versions, { id: "draft", status: "review_required", source_version_id: "old" } as never] }} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Continue draft" }))
  expect(push).toHaveBeenCalledWith("/guidelines/doc/versions/draft/markdown")
  expect(revise).not.toHaveBeenCalled()
})

it.each(["file", "link"])("opens the %s draft controls instead of the Markdown editor", async mode => {
  revise.mockResolvedValue({ version: { id: "draft", version: "2", revision_warnings: [] }, existing: false })
  render(<GuidelineRevisionAction document={{ ...document, document_kind: { publish_as_uploaded: mode === "file", publish_as_link: mode === "link" } as never }} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  fireEvent.change(screen.getByLabelText("New version number"), { target: { value: "2" } })
  fireEvent.click(screen.getByRole("button", { name: "Create revision draft" }))
  fireEvent.click(await screen.findByRole("button", { name: "Open draft" }))
  expect(push).toHaveBeenCalledWith("/guidelines/doc#version-draft")
})

it("keeps failed submissions open with entered version and actionable error", async () => {
  revise.mockRejectedValue(new Error("Publication changed; refresh"))
  render(<GuidelineRevisionAction document={document} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  fireEvent.change(screen.getByLabelText("New version number"), { target: { value: "2" } })
  fireEvent.click(screen.getByRole("button", { name: "Create revision draft" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("Publication changed; refresh")
  expect(screen.getByLabelText("New version number")).toHaveValue("2")
})

it("offers no withdrawal revision action for a current publication", () => {
  render(<GuidelineRevisionAction document={{ ...document, withdrawn_at: null, withdrawn_version_id: null }} onChanged={vi.fn()} />)
  expect(screen.queryByRole("button")).not.toBeInTheDocument()
})

it("suggests a label beyond existing versions in the withdrawn version's series", () => {
  const source = { ...document.versions[0], version: "1.0" }
  render(<GuidelineRevisionAction document={{ ...document, versions: [source, { ...source, id: "later", version: "1.3" }] }} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  expect(screen.getByLabelText("New version number")).toHaveValue("1.4")
})

it("requires a manual label for an unrecognized version format", () => {
  render(<GuidelineRevisionAction document={{ ...document, versions: [{ ...document.versions[0], version: "Official edition A" }] }} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  expect(screen.getByLabelText("New version number")).toHaveValue("")
  expect(screen.getByRole("button", { name: "Create revision draft" })).toBeDisabled()
})

it("rejects a manually entered existing label without creating a draft", () => {
  render(<GuidelineRevisionAction document={document} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Revise document" }))
  fireEvent.change(screen.getByLabelText("New version number"), { target: { value: " 1 " } })
  fireEvent.click(screen.getByRole("button", { name: "Create revision draft" }))
  expect(screen.getByRole("alert")).toHaveTextContent("This version number already exists")
  expect(revise).not.toHaveBeenCalled()
})
