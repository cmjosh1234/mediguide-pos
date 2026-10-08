import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

const documentId = "11111111-1111-4111-8111-111111111111";
const otherDocumentId = "22222222-2222-4222-8222-222222222222";

const reports = vi.hoisted(() => ({
  createAttachment: vi.fn(async () => ({})),
  updateAttachment: vi.fn(async () => ({})),
  removeAttachment: vi.fn(async () => undefined),
}));
const outbreaks = vi.hoisted(() => ({
  listPublishedGuidelines: vi.fn(async () => ({
    items: [
      { id: "11111111-1111-4111-8111-111111111111", title: "Week 12 situation report (PDF)" },
      { id: "22222222-2222-4222-8222-222222222222", title: "Week 12 annex" },
    ],
  })),
}));

vi.mock("@/services/situation-reports.service", () => ({ situationReportsService: reports }));
vi.mock("@/services/outbreaks.service", () => ({ outbreaksService: outbreaks }));
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }));

import { ReportAttachments } from "./report-attachments";

const kinds = [
  { id: "k1", slug: "situation_report_attachment", name: "Situation Report Attachment", description: "Attachments published with situation reports.", sort_order: 1, status: "active" as const, publish_as_uploaded: true, publish_as_link: false, guideline_document_count: 2, outbreak_resource_count: 0 },
  { id: "k2", slug: "sop", name: "SOP", description: "", sort_order: 2, status: "active" as const, publish_as_uploaded: false, publish_as_link: false, guideline_document_count: 0, outbreak_resource_count: 0 },
];
const attached = {
  id: "a1",
  title: "Full report",
  document_kind: "situation_report_attachment",
  url: `/public/guidelines/${documentId}`,
  lock_version: 3,
};

function renderCard(props: Partial<React.ComponentProps<typeof ReportAttachments>> = {}) {
  const onChanged = vi.fn(async () => {});
  render(
    <ReportAttachments
      reportId="r1"
      attachments={[]}
      documentKinds={kinds}
      locked={false}
      hasUploadedPdf={false}
      onChanged={onChanged}
      {...props}
    />,
  );
  return onChanged;
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("adds a published library document, defaulting to the attachment document type", async () => {
  const onChanged = renderCard();
  expect((screen.getByLabelText(/^Document type/) as HTMLSelectElement).value).toBe("situation_report_attachment");
  expect(screen.queryByRole("option", { name: "Situation report" })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Add attachment" }));
  expect(await screen.findByText("Title is required.")).toBeTruthy();
  expect(reports.createAttachment).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText(/^Title/), { target: { value: "Full report" } });
  const picker = await screen.findByLabelText(/^Published Situation Report Attachment/);
  fireEvent.change(picker, { target: { value: `/public/guidelines/${documentId}` } });
  fireEvent.click(screen.getByRole("button", { name: "Add attachment" }));

  await waitFor(() =>
    expect(reports.createAttachment).toHaveBeenCalledWith("r1", {
      title: "Full report",
      description: "",
      issuing_organization: "",
      document_kind: "situation_report_attachment",
      url: `/public/guidelines/${documentId}`,
    }),
  );
  expect(onChanged).toHaveBeenCalled();
});

test("edits and removes an attachment", async () => {
  const onChanged = renderCard({ attachments: [attached] });
  expect(await screen.findByText("Situation Report Attachment · Week 12 situation report (PDF)")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/^Title/), { target: { value: "Full week 12 report" } });
  fireEvent.change(await within(dialog).findByLabelText(/^Published Situation Report Attachment/), {
    target: { value: `/public/guidelines/${otherDocumentId}` },
  });
  fireEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() =>
    expect(reports.updateAttachment).toHaveBeenCalledWith("r1", "a1", expect.objectContaining({
      title: "Full week 12 report",
      url: `/public/guidelines/${otherDocumentId}`,
      lock_version: 3,
    })),
  );

  vi.spyOn(window, "confirm").mockReturnValue(true);
  fireEvent.click(screen.getByRole("button", { name: "Remove" }));
  await waitFor(() => expect(reports.removeAttachment).toHaveBeenCalledWith("r1", "a1", 3));
  expect(onChanged).toHaveBeenCalledTimes(2);
});

test("a published report's attachments are locked", async () => {
  renderCard({ attachments: [attached], locked: true });
  expect(screen.getByText(/Attachments are locked once the report is published/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Add attachment" })).toBeNull();
  expect((screen.getByRole("button", { name: "Edit" }) as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("button", { name: "Remove" }) as HTMLButtonElement).disabled).toBe(true);
});

test("an attachment whose document was unpublished is flagged", async () => {
  outbreaks.listPublishedGuidelines.mockResolvedValueOnce({ items: [] });
  // Locked, so the only lookup is the attachment's own (no add form picker).
  renderCard({ attachments: [attached], locked: true });
  expect(await screen.findByText(/no longer published/)).toBeTruthy();
});
