import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { DecisionToolForm } from "./decision-tool-form"
import { CalculatorsStatusOptions, type CalculatorsResponse } from "@/types/backend-types"

const permissions = vi.hoisted(() => ({ reviewer: false }))
vi.mock("@/lib/backend-client", () => ({
  hasBackendPermission: (permission: string) => permission === "calculator.review" && permissions.reviewer,
}))
vi.mock("@/hooks/use-tool-categories", () => {
  const categories = [
    { id: "cat-1", name: "Neurology", slug: "neurology", description: "", sort_order: 0, status: "active", color: "", icon: "brain", tool_count: 0, created: "", updated: "" },
  ]
  return {
    useToolCategories: () => ({
      categories,
      activeCategories: categories,
      byId: new Map(categories.map((category) => [category.id, category])),
      loading: false,
      error: null,
    }),
  }
})

beforeEach(() => {
  permissions.reviewer = false
})
afterEach(cleanup)

const noop = async () => {}

function renderForm(
  mode: "create" | "edit",
  initialData?: Partial<CalculatorsResponse> & { current_version_id?: string | null },
  onSubmit: (data: Record<string, unknown>) => Promise<void> = noop,
) {
  render(<DecisionToolForm mode={mode} initialData={initialData} onSubmit={onSubmit} onCancel={() => {}} />)
}

describe("DecisionToolForm", () => {
  it("does not offer the old priority field", () => {
    renderForm("create")
    expect(screen.queryByText(/Priority/)).toBeNull()
  })

  it("requires a category before saving", async () => {
    const onSubmit = vi.fn<(data: Record<string, unknown>) => Promise<void>>().mockResolvedValue(undefined)
    renderForm("create", { name: "GCS", version: "1.0.0" }, onSubmit)
    fireEvent.click(screen.getByRole("button", { name: /create/i }))
    expect(await screen.findByText("Choose a category")).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it("submits the category and risk level", async () => {
    const onSubmit = vi.fn<(data: Record<string, unknown>) => Promise<void>>().mockResolvedValue(undefined)
    renderForm("create", { name: "GCS", version: "1.0.0", category_id: "cat-1", risk_level: "high" }, onSubmit)
    fireEvent.click(screen.getByRole("button", { name: /create/i }))
    await waitFor(() => expect(onSubmit).toHaveBeenCalled())
    expect(onSubmit.mock.calls[0][0]).toMatchObject({ category_id: "cat-1", risk_level: "high" })
  })

  it("shows the review rule for the chosen risk level", () => {
    renderForm("create", { risk_level: "critical" })
    expect(screen.getByText(/A different person must approve and publish each version./)).toBeInTheDocument()
  })

  it("tells non-reviewers they cannot lower an existing tool's risk", () => {
    renderForm("edit", { risk_level: "high" })
    expect(screen.getByText(/Only clinical reviewers can lower/)).toBeInTheDocument()
  })

  it("lets reviewers lower risk without the notice", () => {
    permissions.reviewer = true
    renderForm("edit", { risk_level: "high" })
    expect(screen.queryByText(/Only clinical reviewers can lower/)).toBeNull()
  })

  it("warns when a new tool is set to Active", () => {
    renderForm("create", { status: CalculatorsStatusOptions.active })
    expect(screen.getByRole("alert")).toHaveTextContent("no published version")
  })

  it("does not warn for a Draft tool", () => {
    renderForm("create")
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("warns when an edited tool without a published version is Active", () => {
    renderForm("edit", { status: CalculatorsStatusOptions.active, current_version_id: null })
    expect(screen.getByRole("alert")).toBeInTheDocument()
  })

  it("does not warn when the edited tool has a published version", () => {
    renderForm("edit", { status: CalculatorsStatusOptions.active, current_version_id: "version-1" })
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("warns for a duplicate even if the source tool was published", () => {
    renderForm("create", { status: CalculatorsStatusOptions.active, current_version_id: "version-1" })
    expect(screen.getByRole("alert")).toBeInTheDocument()
  })
})
