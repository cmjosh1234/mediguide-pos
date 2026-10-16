import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import type { GuidelineDocumentRecord } from "@/services/guideline-documents.service"
import { GuidelinePublicationAction } from "./guideline-publication-action"

const { withdraw, publish, success, error } = vi.hoisted(() => ({
  withdraw: vi.fn(), publish: vi.fn(), success: vi.fn(), error: vi.fn(),
}))
vi.mock("@/services/guideline-documents.service", () => ({
  GuidelineDocumentsService: { withdrawDocument: withdraw, publishVersion: publish },
}))
vi.mock("@/lib/toast", () => ({ showToast: { success, error } }))

const document = {
  id: "document-1", title: "Clinical guidance", current_version_id: "version-1",
  versions: [{ id: "version-1", version: "1", status: "published" }],
} as GuidelineDocumentRecord

beforeEach(() => vi.resetAllMocks())
afterEach(cleanup)

it("requires a reason and withdraws the expected version", async () => {
  const onChanged = vi.fn().mockResolvedValue(undefined)
  render(<GuidelinePublicationAction document={document} onChanged={onChanged} />)
  fireEvent.click(screen.getByRole("button", { name: "Withdraw" }))
  expect(screen.getByRole("button", { name: "Withdraw document" })).toBeDisabled()
  fireEvent.change(screen.getByLabelText("Reason for withdrawal"), { target: { value: "  Source review  " } })
  fireEvent.click(screen.getByRole("button", { name: "Withdraw document" }))
  await waitFor(() => expect(withdraw).toHaveBeenCalledWith("document-1", "version-1", "Source review"))
  expect(onChanged).toHaveBeenCalled()
})

it("keeps a failed withdrawal open with the reason intact", async () => {
  withdraw.mockRejectedValue(new Error("Refresh and try again"))
  render(<GuidelinePublicationAction document={document} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Withdraw" }))
  fireEvent.change(screen.getByLabelText("Reason for withdrawal"), { target: { value: "Source review" } })
  fireEvent.click(screen.getByRole("button", { name: "Withdraw document" }))
  await waitFor(() => expect(error).toHaveBeenCalledWith("Withdrawal failed", "Refresh and try again"))
  expect(screen.getByLabelText("Reason for withdrawal")).toHaveValue("Source review")
})

it("requires review confirmation before republishing the retained version", async () => {
  render(<GuidelinePublicationAction document={{ ...document, current_version_id: null, withdrawn_version_id: "version-1", withdrawn_at: "2026-10-16T09:00:00Z", withdrawal_reason: "Source review" }} onChanged={vi.fn()} />)
  fireEvent.click(screen.getByRole("button", { name: "Review and republish" }))
  expect(screen.getByRole("button", { name: "Republish" })).toBeDisabled()
  expect(screen.getByText(/Source review/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole("checkbox"))
  fireEvent.click(screen.getByRole("button", { name: "Republish" }))
  await waitFor(() => expect(publish).toHaveBeenCalledWith("version-1"))
})
