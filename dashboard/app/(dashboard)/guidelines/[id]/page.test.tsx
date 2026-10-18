import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import GuidelineDetailsPage from "./page"
import type { GuidelineDocumentRecord } from "@/services/guideline-documents.service"

const { state, push } = vi.hoisted(() => ({ state: { document: {} as GuidelineDocumentRecord }, push: vi.fn() }))
vi.mock("next/navigation", () => ({ useParams: () => ({ id: "doc" }), useRouter: () => ({ push, replace: vi.fn() }) }))
vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: string[] }) => ({ data: queryKey.length === 2 ? state.document : queryKey.includes("sections") ? [] : "", isLoading: false, isError: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn().mockResolvedValue(undefined) }),
}))
vi.mock("@/lib/permission-context", () => ({ usePermissionContext: () => ({ hasPermission: () => true, loading: false }) }))
vi.mock("@/lib/backend-client", () => ({ hasBackendPermission: () => true }))
vi.mock("../components/guideline-version-dialogs", () => ({ CreateVersionDialog: () => null, UploadVersionDialog: () => null, VersionLinkDialog: () => null }))
vi.mock("@/components/guidelines/original-file-preview", () => ({ OriginalFilePreview: () => <p>Retained original preview</p> }))
beforeEach(() => {
  state.document = {
    id: "doc", created_at: "", updated_at: "", title: "Withdrawn guidance", withdrawn_at: "2026-10-16T12:00:00Z", withdrawn_version_id: "source", withdrawal_reason: "Clinical correction", current_version_id: null,
    versions: [{ id: "source", document_id: "doc", created_at: "", updated_at: "", status: "published", version: "1", original_file_key: "source.pdf", markdown_file_key: "source.md", html_file_key: "source.html" }], categories: [],
  }
})
afterEach(cleanup)
it("keeps the withdrawn version read-only and exposes revision, republish and original inspection", () => {
  render(<GuidelineDetailsPage />)
  expect(screen.getByRole("button", { name: "Revise document" })).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "Review and republish" })).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "Edit details" })).toBeInTheDocument()
  for (const name of ["Upload PDF", "Edit Markdown", "Publish", "New Version"]) expect(screen.queryByRole("button", { name })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole("button", { name: "Preview original" }))
  expect(screen.getByText("Retained original preview")).toBeInTheDocument()
})
it("keeps a separate revision draft editable while its source stays withdrawn", () => {
  state.document.versions.push({ id: "draft", document_id: "doc", created_at: "", updated_at: "", status: "draft", version: "2", source_version_id: "source", markdown_file_key: "draft.md", structured_content_status: "outdated" })
  render(<GuidelineDetailsPage />)
  expect(screen.getByRole("button", { name: "Continue draft" })).toBeInTheDocument()
  expect(screen.getAllByRole("button", { name: "Upload PDF" })).toHaveLength(1)
  expect(screen.getAllByRole("button", { name: "Edit Markdown" })).toHaveLength(1)
  expect(screen.queryByRole("button", { name: "Publish" })).not.toBeInTheDocument()
})
