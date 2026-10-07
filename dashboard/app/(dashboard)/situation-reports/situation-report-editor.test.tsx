import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { BackendRequestError } from "@/lib/backend-client";

const draft = {
  id: "r1",
  title: "Week 12 situation report",
  status: "draft",
  standalone_allowed: true,
  geographic_area: "Gulu",
  source_organization: "Ministry of Health",
  source_reference: "",
  publication_date: "2026-07-01T00:00:00Z",
  effective_at: "2026-06-30T00:00:00Z",
  last_verified_at: "2026-07-01T00:00:00Z",
  lock_version: 1,
  metrics: [],
  key_highlights: [],
};

const reports = vi.hoisted(() => ({
  get: vi.fn(),
  audit: vi.fn(async () => ({ items: [] })),
  listAttachments: vi.fn(async (): Promise<Record<string, unknown>[]> => []),
  transition: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(async () => undefined),
  updateMetrics: vi.fn(),
}));

vi.mock("@/services/situation-reports.service", () => ({ situationReportsService: reports }));
vi.mock("@/services/outbreaks.service", () => ({
  outbreaksService: {
    list: async () => ({ items: [] }),
    listPublishedGuidelines: async () => ({ items: [] }),
  },
}));
vi.mock("@/services/document-kinds.service", () => ({ documentKindService: { list: async () => [] } }));
vi.mock("@/services/health-facilities.service", () => ({ healthFacilitiesService: { regions: async () => [], districts: async () => [] } }));
const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@/lib/toast", () => ({ showToast: toast }));
vi.mock("@/lib/backend-client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/backend-client")>()),
  getCurrentUser: () => ({ id: "user-b" }),
}));

import { SituationReportEditor } from "./situation-report-editor";

beforeEach(() => {
  vi.clearAllMocks();
  reports.get.mockResolvedValue(draft);
  Element.prototype.scrollIntoView = vi.fn();
});
afterEach(cleanup);

test("the publication workflow comes first and explains what publishing still needs", async () => {
  render(<SituationReportEditor id="r1" />);
  const workflow = await screen.findByText("Publication workflow");
  const metadata = screen.getByText("Report metadata");
  expect(workflow.compareDocumentPosition(metadata) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

  expect(screen.getByText(/Fill in the details, add attachments and save the draft/)).toBeTruthy();
  expect(screen.getByText("Before it can be published, fill in and save: Source reference.")).toBeTruthy();
  expect(screen.getByText("Before it can be published, add at least one attachment.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Submit for review" }) as HTMLButtonElement).disabled).toBe(false);
  expect((screen.getByRole("button", { name: "Publish" }) as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("button", { name: "Withdraw" }) as HTMLButtonElement).disabled).toBe(true);
});

test("a failed workflow step highlights the fields to fix", async () => {
  const reason = "Publication date can't be earlier than Effective at.";
  reports.transition.mockRejectedValue(
    new BackendRequestError(reason, 400, undefined, {
      fields: [
        { field: "publication_date", message: reason },
        { field: "effective_at", message: reason },
      ],
    }),
  );
  render(<SituationReportEditor id="r1" />);
  const published = await screen.findByLabelText(/^Publication date/);
  const effective = screen.getByLabelText(/^Effective at/);

  fireEvent.click(screen.getByRole("button", { name: "Submit for review" }));
  await waitFor(() => expect(published.getAttribute("aria-invalid")).toBe("true"));
  expect(effective.getAttribute("aria-invalid")).toBe("true");
  expect(screen.getAllByText(reason)).toHaveLength(2);

  fireEvent.change(effective, { target: { value: "2026-06-29T00:00" } });
  expect(published.getAttribute("aria-invalid")).toBe("false");
});

test("unsaved changes must be saved before submitting", async () => {
  reports.update.mockImplementation(async (_id: string, input: Record<string, unknown>) => ({ ...draft, ...input, lock_version: 2 }));
  render(<SituationReportEditor id="r1" />);
  const reference = await screen.findByLabelText(/^Source reference/);
  const submit = screen.getByRole("button", { name: "Submit for review" }) as HTMLButtonElement;

  fireEvent.change(reference, { target: { value: "MOH-SITREP-12" } });
  expect(screen.getByText(/save the draft first/)).toBeTruthy();
  expect(submit.disabled).toBe(true);

  fireEvent.click(screen.getAllByRole("button", { name: "Save draft" })[0]);
  await waitFor(() => expect(screen.queryByText(/save the draft first/)).toBeNull());
  expect(submit.disabled).toBe(false);
  // The saved report has its source reference, so only the attachment is missing.
  expect(screen.queryByText(/fill in and save/)).toBeNull();
});

test("a published report can be corrected or withdrawn, and its fields are locked", async () => {
  reports.get.mockResolvedValue({ ...draft, status: "published", source_reference: "MOH-SITREP-12", approved_at: "2026-07-01T00:00:00Z", published_at: "2026-07-01T00:00:00Z" });
  render(<SituationReportEditor id="r1" />);
  expect(await screen.findByText(/Live on the public API/)).toBeTruthy();
  expect(screen.getByText("Published reports are immutable.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Create correction" }) as HTMLButtonElement).disabled).toBe(false);
  expect((screen.getByRole("button", { name: "Withdraw" }) as HTMLButtonElement).disabled).toBe(false);
  expect((screen.getByRole("button", { name: "Submit for review" }) as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByLabelText(/^Title/) as HTMLInputElement).disabled).toBe(true);
});

test("whoever submitted a report can't approve it", async () => {
  reports.get.mockResolvedValue({ ...draft, status: "pending_review", author_id: "user-a", submitted_by: "user-b" });
  render(<SituationReportEditor id="r1" />);
  expect(await screen.findByText("You submitted this report for review, so a different reviewer has to approve it.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Approve" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByText(/must be someone other than its author and the person who submitted it/)).toBeTruthy();
});

test("someone other than the author and submitter can approve", async () => {
  reports.get.mockResolvedValue({ ...draft, status: "pending_review", author_id: "user-a", submitted_by: "user-c" });
  render(<SituationReportEditor id="r1" />);
  expect(await screen.findByText(/Waiting for a reviewer/)).toBeTruthy();
  expect((screen.getByRole("button", { name: "Approve" }) as HTMLButtonElement).disabled).toBe(false);
  expect(screen.queryByText(/so a different reviewer has to approve it/)).toBeNull();
});

const published = { ...draft, id: "r0", title: "Week 12 situation report", status: "published", source_reference: "MOH-SITREP-12", approved_at: "2026-07-01T00:00:00Z", published_at: "2026-07-01T00:00:00Z", lock_version: 5 };
const correction = { ...published, id: "r1", status: "pending_review", supersedes_id: "r0", correction_reason: "Wrong week", author_id: "user-a", submitted_by: "user-c", approved_at: undefined, published_at: undefined, lock_version: 3 };

test("a correction is approved and applied, or discarded, instead of published", async () => {
  reports.get.mockImplementation(async (id: string) => (id === "r0" ? published : correction));
  reports.listAttachments.mockResolvedValue([{ id: "a1", title: "Full report", document_kind: "situation_report_attachment", url: "/public/guidelines/11111111-1111-4111-8111-111111111111", lock_version: 1 }]);
  render(<SituationReportEditor id="r1" />);

  expect(await screen.findByText("Correction workflow")).toBeTruthy();
  const link = await screen.findByRole("link", { name: "Week 12 situation report" });
  expect(link.getAttribute("href")).toBe("/situation-reports/r0");
  expect(screen.getByText(/Approving replaces the published report with this version/)).toBeTruthy();
  expect(screen.getByText("Applied")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Publish" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Create correction" })).toBeNull();
  expect((screen.getByLabelText(/^Related outbreak/) as HTMLSelectElement).disabled).toBe(true);

  reports.transition.mockResolvedValue(published);
  vi.spyOn(window, "confirm").mockReturnValue(true);
  fireEvent.click(screen.getByRole("button", { name: "Approve and apply" }));
  await waitFor(() => expect(reports.transition).toHaveBeenCalledWith("r1", "approve", expect.objectContaining({ lock_version: 3 })));
  await waitFor(() => expect(toast.success).toHaveBeenCalledWith("Correction applied", expect.any(String)));

  fireEvent.click(screen.getByRole("button", { name: "Discard correction" }));
  await waitFor(() => expect(reports.remove).toHaveBeenCalledWith("r1", 3));
});

test("a new report is saved from the end of the form, not the page header", async () => {
  render(<SituationReportEditor />);
  await screen.findByText("New situation report");
  const header = screen.getByRole("link", { name: "Back to list" }).parentElement!;
  expect(within(header).queryByRole("button", { name: "Save draft" })).toBeNull();
  expect(screen.getAllByRole("button", { name: "Save draft" })).toHaveLength(1);
});

test("a correction is saved from its workflow card, not the page header", async () => {
  reports.get.mockImplementation(async (id: string) => (id === "r0" ? published : correction));
  render(<SituationReportEditor id="r1" />);
  const reference = await screen.findByLabelText(/^Source reference/);
  const header = screen.getByRole("link", { name: "Back to list" }).parentElement!;
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();

  fireEvent.change(reference, { target: { value: "MOH-SITREP-13" } });
  expect(within(header).queryByRole("button", { name: "Save draft" })).toBeNull();
  const card = screen.getByText("Correction workflow").closest("[data-slot=card]") as HTMLElement;
  expect(within(card).getByRole("button", { name: "Save draft" })).toBeTruthy();
});

test("a published report points to its open correction", async () => {
  reports.get.mockResolvedValue({ ...published, id: "r1", open_correction_id: "c9" });
  render(<SituationReportEditor id="r1" />);
  const link = await screen.findByRole("link", { name: "Open the correction" });
  expect(link.getAttribute("href")).toBe("/situation-reports/c9");
  expect((screen.getByRole("button", { name: "Create correction" }) as HTMLButtonElement).disabled).toBe(true);
});

test("metrics are added through the dialog and saved straight away", async () => {
  reports.get.mockResolvedValue(draft);
  reports.updateMetrics.mockImplementation(async (_id: string, input: { metrics: unknown[] }) => ({ ...draft, metrics: input.metrics, lock_version: 2 }));
  render(<SituationReportEditor id="r1" />);

  fireEvent.click(await screen.findByRole("button", { name: "Metric" }));
  const dialog = await screen.findByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/^Label/), { target: { value: "Confirmed cases" } });
  // The key follows the label.
  expect((within(dialog).getByLabelText(/^Key/) as HTMLInputElement).value).toBe("confirmed_cases");
  fireEvent.change(within(dialog).getByLabelText(/^Value/), { target: { value: "20" } });
  fireEvent.change(within(dialog).getByLabelText(/^Source/), { target: { value: "WHO situation report 11" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Save metric" }));

  await waitFor(() =>
    expect(reports.updateMetrics).toHaveBeenCalledWith("r1", {
      metrics: [expect.objectContaining({ key: "confirmed_cases", label: "Confirmed cases", value: "20", source_reference: "WHO situation report 11", sort_order: 1 })],
      lock_version: 1,
    }),
  );
  expect(await screen.findByText("confirmed_cases")).toBeTruthy();
  expect(screen.queryByRole("dialog")).toBeNull();
  // Saving metrics doesn't leave the form with unsaved changes.
  expect(screen.queryByText(/save the draft first/)).toBeNull();
});

test("a published report's metrics are locked", async () => {
  // No attachments, so the only Remove buttons would be the metrics'.
  reports.listAttachments.mockResolvedValue([]);
  reports.get.mockResolvedValue({
    ...draft,
    status: "published",
    source_reference: "MOH-SITREP-12",
    published_at: "2026-07-01T00:00:00Z",
    metrics: [{ key: "deaths", label: "Deaths", value: "2", unit: "deaths", as_of: "2026-07-01T00:00:00Z", source_reference: "WHO", sort_order: 1 }],
  });
  render(<SituationReportEditor id="r1" />);
  expect(await screen.findByText("Metrics are locked once the report is published. Create a correction to change them.")).toBeTruthy();
  expect(screen.getByText("deaths")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Metric" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.queryByRole("button", { name: "Remove" })).toBeNull();
});
