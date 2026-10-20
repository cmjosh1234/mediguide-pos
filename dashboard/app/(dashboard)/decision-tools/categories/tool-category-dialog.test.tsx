import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { ToolCategoryDialog } from "./tool-category-dialog"

afterEach(cleanup)

describe("ToolCategoryDialog", () => {
  it("submits a new category and leaves the slug to the backend", async () => {
    const onSubmit = vi.fn<(input: unknown) => Promise<void>>().mockResolvedValue(undefined)
    render(<ToolCategoryDialog open onOpenChange={() => {}} onSubmit={onSubmit} />)
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Maternal & Newborn" } })
    fireEvent.click(screen.getByRole("button", { name: "Add category" }))
    await waitFor(() => expect(onSubmit).toHaveBeenCalled())
    expect(onSubmit.mock.calls[0][0]).toMatchObject({ name: "Maternal & Newborn", slug: undefined, status: "active" })
  })

  it("rejects an invalid colour", async () => {
    const onSubmit = vi.fn<(input: unknown) => Promise<void>>().mockResolvedValue(undefined)
    render(<ToolCategoryDialog open onOpenChange={() => {}} onSubmit={onSubmit} />)
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Neurology" } })
    fireEvent.change(screen.getByLabelText("Colour"), { target: { value: "blue" } })
    fireEvent.click(screen.getByRole("button", { name: "Add category" }))
    expect(await screen.findByText(/Use a hex colour/)).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it("prefills an existing category for editing", () => {
    render(
      <ToolCategoryDialog
        open
        onOpenChange={() => {}}
        onSubmit={async () => {}}
        category={{ id: "c1", name: "Neurology", slug: "neurology", description: "", sort_order: 60, status: "active", color: "#4527A0", icon: "brain", tool_count: 1, created: "", updated: "" }}
      />,
    )
    expect(screen.getByLabelText("Name *")).toHaveValue("Neurology")
    expect(screen.getByLabelText("Icon")).toHaveTextContent("Brain")
    expect(screen.getByRole("button", { name: "Save changes" })).toBeInTheDocument()
  })
})
