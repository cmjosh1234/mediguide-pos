import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

const reports = vi.hoisted(() => ({ audit: vi.fn(), addReviewComment: vi.fn(async () => undefined) }));

vi.mock("@/services/outbreaks.service", () => ({ outbreaksService: {} }));
vi.mock("@/services/situation-reports.service", () => ({ situationReportsService: reports }));
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }));

import { AuditHistory } from "./audit-history";

const amina = "321391ea-7370-4a70-9bf8-e1a27095d3c3", brian = "a3680cf8-0650-4299-8dcc-3b017df717bc";
const kabale = "bcef5f36-a79e-5dd9-a26e-a5cf5e597a5a", rukungiri = "6a33f840-f995-5348-8a6b-fe3b8309dce7";

const applied = {
  id: "e1", actor_id: brian, actor_name: "Brian Mugisha", actor_email: "brian@example.test",
  action: "situation_report.correction_applied", entity_type: "situation_report", entity_id: "r1", created_at: "2026-10-07T14:25:17Z",
  metadata: { reason: "Wrong district", corrected_by: amina, correction_id: "c1", changes: { district_id: { from: kabale, to: rukungiri } } },
  labels: { [amina]: "Amina Okello", [kabale]: "Kabale", [rukungiri]: "Rukungiri" },
};
const statusChange = {
  ...applied, id: "e9", actor_id: amina, actor_name: "Amina Okello", actor_email: "amina@example.test",
  action: "situation_report.publish", metadata: { reason: "", from_status: "pending_review", to_status: "published" }, labels: undefined,
};

function page(items: unknown[], number: number, total: number) {
  return { items, page: number, per_page: 5, total_items: total, total_pages: Math.ceil(total / 5) };
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("each entry says what was done, by whom, and what changed, without raw IDs", async () => {
  reports.audit.mockResolvedValue(page([applied], 1, 1));
  render(<AuditHistory entity="situation_report" id="r1" />);

  const entry = (await screen.findByText("Correction applied")).closest("li")!;
  const text = entry.textContent || "";
  expect(text).toContain("by Brian Mugisha · brian@example.test");
  expect(text).toContain("Wrong district");
  expect(text).toContain("Correction byAmina Okello");
  expect(text).toContain("DistrictKabale → Rukungiri");
  for (const id of [amina, brian, kabale, rukungiri]) expect(text).not.toContain(id);
});

test("history is paged by the server", async () => {
  const first = Array.from({ length: 5 }, (_, index) => ({ ...applied, id: `p${index}` }));
  reports.audit.mockImplementation(async (_id: string, number: number) => (number === 1 ? page(first, 1, 6) : page([statusChange], 2, 6)));
  render(<AuditHistory entity="situation_report" id="r1" />);

  expect(await screen.findByText("Showing 1–5 of 6")).toBeTruthy();
  expect(reports.audit).toHaveBeenLastCalledWith("r1", 1, 5);
  fireEvent.click(screen.getByRole("button", { name: "Next" }));

  expect(await screen.findByText("Published")).toBeTruthy();
  expect(reports.audit).toHaveBeenLastCalledWith("r1", 2, 5);
  expect(screen.getByText("Showing 6–6 of 6")).toBeTruthy();
  expect(screen.getByText("Pending review → Published")).toBeTruthy();
});

test("a review comment is recorded and the history reloads from the first page", async () => {
  reports.audit.mockImplementation(async (_id: string, number: number) => page([{ ...applied, id: `n${number}` }], number, 6));
  render(<AuditHistory entity="situation_report" id="r1" />);
  fireEvent.click(await screen.findByRole("button", { name: "Next" }));
  await waitFor(() => expect(reports.audit).toHaveBeenLastCalledWith("r1", 2, 5));

  fireEvent.change(screen.getByPlaceholderText("Add an auditable reviewer comment"), { target: { value: "  Checked against WHO sitrep 11 " } });
  fireEvent.click(screen.getByRole("button", { name: "Add comment" }));

  await waitFor(() => expect(reports.addReviewComment).toHaveBeenCalledWith("r1", "Checked against WHO sitrep 11"));
  await waitFor(() => expect(reports.audit).toHaveBeenLastCalledWith("r1", 1, 5));
});
