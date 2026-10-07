import * as React from "react"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, describe, expect, it, vi } from "vitest"
import { DataTable } from "./data-table"
import { BackendDataTable } from "./backend-data-table"
import type { BaseRecord } from "@/types/data-table"

vi.mock("@/lib/toast", () => ({ showToast: { error: vi.fn() } }))
afterEach(cleanup)

describe("empty table recovery", () => {
  it("clears a client-side search and restores the existing records", () => {
    render(<DataTable columns={[{ accessorKey: "name", header: "Name" }]} data={[{ name: "Malaria" }]} searchKey="name" />)
    fireEvent.change(screen.getByPlaceholderText("Filter..."), { target: { value: "missing" } })
    expect(screen.getByText("No matching results")).toBeVisible()
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }))
    expect(screen.getByText("Malaria")).toBeVisible()
    expect(screen.getByPlaceholderText("Filter...")).toHaveValue("")
  })

  it("distinguishes an empty collection from an unsuccessful search", () => {
    render(<DataTable columns={[{ accessorKey: "name", header: "Name" }]} data={[]} searchKey="name" />)
    expect(screen.getByText("No records yet")).toBeVisible()
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument()
  })

  it("clears server-side search and the toolbar together", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const record: BaseRecord = { id: "one", created: "", updated: "", name: "Malaria" }
    const loadPage = vi.fn(async ({ page, perPage, search }: { page: number; perPage: number; search: string }) => ({
      items: search ? [] : [record], totalItems: search ? 0 : 1, page, perPage, totalPages: search ? 0 : 1,
    }))
    render(<QueryClientProvider client={client}><BackendDataTable collection="diseases" columns={[{ accessorKey: "name", header: "Name" }]} loadPage={loadPage} searchFields={["name"]} ui={{ exportable: false, importable: false }} /></QueryClientProvider>)
    expect(screen.queryByText("No diseases yet")).not.toBeInTheDocument()
    await screen.findByText("Malaria")
    fireEvent.change(screen.getByPlaceholderText("Search..."), { target: { value: "missing" } })
    await screen.findByText("No matching results")
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }))
    await screen.findByText("Malaria")
    await waitFor(() => expect(screen.getByPlaceholderText("Search...")).toHaveValue(""))
    client.clear()
  })

  it("shows a failed request as an error and allows retrying it", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const loadPage = vi.fn()
      .mockRejectedValueOnce(new Error("Connection unavailable"))
      .mockResolvedValue({ items: [], totalItems: 0, page: 1, perPage: 20, totalPages: 0 })
    render(<QueryClientProvider client={client}><BackendDataTable collection="diseases" columns={[{ accessorKey: "name", header: "Name" }]} loadPage={loadPage} ui={{ exportable: false, importable: false }} /></QueryClientProvider>)
    await screen.findByText("Connection unavailable")
    expect(screen.queryByText("No diseases yet")).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))
    await screen.findByText("No diseases yet")
    expect(loadPage).toHaveBeenCalledTimes(2)
    client.clear()
  })
})
