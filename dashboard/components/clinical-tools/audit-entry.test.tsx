import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"
import { ClinicalToolAuditEntryItem } from "./audit-entry"

afterEach(cleanup)

describe("ClinicalToolAuditEntryItem", () => {
  it("shows what happened and who did it", () => {
    render(
      <ClinicalToolAuditEntryItem
        item={{ id: "a1", action: "calculator.version.submitted", created_at: "2026-10-20T15:14:56Z", actor_id: "u1", actor_name: "Grace Author", actor_email: "grace@example.org", from_status: "draft", to_status: "pending_review" }}
      />,
    )
    expect(screen.getByText("Submitted for review")).toBeInTheDocument()
    expect(screen.getByText("Grace Author")).toBeInTheDocument()
    expect(screen.getByText(/grace@example.org/)).toBeInTheDocument()
    expect(screen.getByText("Status")).toBeInTheDocument()
  })

  it("shows review comments and skips unchanged statuses", () => {
    render(
      <ClinicalToolAuditEntryItem
        item={{ id: "a2", action: "calculator.version.review_commented", created_at: "2026-10-20T15:20:00Z", actor_id: "u2", from_status: "pending_review", to_status: "pending_review", metadata: { comment: "Check the units." } }}
      />,
    )
    expect(screen.getByText("Review comment")).toBeInTheDocument()
    expect(screen.getByText("Check the units.")).toBeInTheDocument()
    expect(screen.queryByText("Status")).toBeNull()
    expect(screen.getByText(/Unknown user/)).toBeInTheDocument()
  })
})
