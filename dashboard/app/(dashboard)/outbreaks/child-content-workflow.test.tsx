import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

const service = vi.hoisted(() => ({
  editPublishedResource: vi.fn(async () => ({})),
  updateResource: vi.fn(async () => ({})),
  deleteResource: vi.fn(async () => undefined),
  editPublishedUpdate: vi.fn(async () => ({})),
  updateUpdate: vi.fn(async () => ({})),
  deleteUpdate: vi.fn(async () => undefined),
  listPublishedGuidelines: vi.fn(async () => ({ items: [{ id: "g1", title: "IPC SOP v2" }, { id: "g2", title: "IPC SOP v3" }] })),
}));

vi.mock("@/services/outbreaks.service", () => ({ outbreaksService: service }));
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }));

import { ChildContentWorkflow } from "./child-content-workflow";

const kinds = [{ id: "k1", slug: "sop", name: "SOP", description: "Standard operating procedures.", sort_order: 1, status: "active" as const, publish_as_uploaded: false, publish_as_link: false, guideline_document_count: 2, outbreak_resource_count: 0 }];
const base = { outbreak_id: "o1", resource_type: "guideline", document_kind: "sop", issuing_organization: "", description: "", sort_order: 1 };

function renderList(items: Record<string, unknown>[], onChanged = vi.fn(async () => {})) {
  render(<ChildContentWorkflow outbreakId="o1" kind="resource" empty="none" items={items} documentKinds={kinds} onChanged={onChanged} />);
  return onChanged;
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("retired versions are hidden and a pending edit is nested under the live resource", async () => {
  const onChanged = renderList([
    { ...base, id: "old", title: "IPC SOP (2023)", url: "/public/guidelines/g0", status: "withdrawn", withdrawal_reason: "superseded by approved correction: newer edition", lock_version: 3 },
    { ...base, id: "live", title: "IPC SOP", url: "/public/guidelines/g1", status: "published", supersedes_id: "old", lock_version: 2 },
    { ...base, id: "edit", title: "IPC SOP (2025)", url: "/public/guidelines/g2", status: "pending_review", supersedes_id: "live", lock_version: 1 },
  ]);
  expect(screen.queryByText("IPC SOP (2023)")).toBeNull();
  expect(screen.getByText("Edit pending review")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Edit pending change" })).toBeTruthy();

  vi.spyOn(window, "confirm").mockReturnValue(true);
  fireEvent.click(screen.getByRole("button", { name: "Discard" }));
  await waitFor(() => expect(service.deleteResource).toHaveBeenCalledWith("o1", "edit", 1));
  expect(onChanged).toHaveBeenCalled();
});

test("editing a published resource requires a reason and submits the edited fields for review", async () => {
  renderList([{ ...base, id: "live", title: "IPC SOP", url: "/public/guidelines/g1", status: "published", lock_version: 2 }]);
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog");
  expect(within(dialog).getByText("Edit published resource")).toBeTruthy();

  const picker = (await within(dialog).findByLabelText(/Published SOP/)) as HTMLSelectElement;
  fireEvent.change(picker, { target: { value: "/public/guidelines/g2" } });
  fireEvent.change(within(dialog).getByLabelText(/^Title/), { target: { value: "IPC SOP (2025)" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Submit edit for review" }));
  expect(await within(dialog).findByText("A reason is required.")).toBeTruthy();
  expect(service.editPublishedResource).not.toHaveBeenCalled();

  fireEvent.change(within(dialog).getByLabelText(/Reason for this change/), { target: { value: "Newer edition" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Submit edit for review" }));
  await waitFor(() => expect(service.editPublishedResource).toHaveBeenCalled());
  expect(service.editPublishedResource).toHaveBeenCalledWith("o1", "live", {
    lock_version: 2,
    reason: "Newer edition",
    changes: expect.objectContaining({ title: "IPC SOP (2025)", resource_type: "guideline", document_kind: "sop", url: "/public/guidelines/g2" }),
  });
});

test("editing a draft saves the changes in place", async () => {
  renderList([{ ...base, id: "draft", title: "IPC SOP", url: "/public/guidelines/g1", status: "draft", lock_version: 4 }]);
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog");
  await within(dialog).findByLabelText(/Published SOP/);
  fireEvent.change(within(dialog).getByLabelText(/^Description/), { target: { value: "Updated description" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(service.updateResource).toHaveBeenCalledWith("o1", "draft", expect.objectContaining({ description: "Updated description", lock_version: 4 })));
  expect(service.editPublishedResource).not.toHaveBeenCalled();
});

const G1 = "11111111-1111-4111-8111-111111111111";
const R1 = "22222222-2222-4222-8222-222222222222";

async function openDetails(title: string) {
  fireEvent.click(screen.getByRole("button", { name: `View details of ${title}` }));
  return screen.findByRole("dialog");
}

test("a linked library document opens on its guideline page", async () => {
  renderList([{ ...base, id: "live", title: "IPC SOP", url: `/public/guidelines/${G1}`, status: "published", lock_version: 1 }]);
  const dialog = await openDetails("IPC SOP");
  const view = within(dialog).getByRole("link", { name: /View document/ });
  expect(view.getAttribute("href")).toContain(`/guidelines/${G1}`);
  expect(view.getAttribute("target")).toBe("_blank");
  expect(within(dialog).queryByRole("button", { name: /Download/ })).toBeNull();
});

test("a linked situation report opens on its report page", async () => {
  renderList([{ ...base, id: "rep", resource_type: "situation_report", document_kind: "other", title: "Week 12 report", url: `/situation-reports/${R1}`, status: "published", lock_version: 1 }]);
  const dialog = await openDetails("Week 12 report");
  expect(within(dialog).getByRole("link", { name: /View report/ }).getAttribute("href")).toContain(`/situation-reports/${R1}`);
  expect(within(dialog).queryByRole("link", { name: /Download/ })).toBeNull();
});

test("internal routes have nothing to open from the dashboard", async () => {
  renderList([{ ...base, id: "route", resource_type: "internal_route", document_kind: "other", title: "Hub", url: "/outbreak-hub", status: "published", lock_version: 1 }]);
  const dialog = await openDetails("Hub");
  expect(within(dialog).queryByRole("link")).toBeNull();
});

test("resources can be paged five at a time", async () => {
  const items = Array.from({ length: 7 }, (_, index) => ({ ...base, id: `r${index + 1}`, title: `Resource ${index + 1}`, url: `/public/guidelines/g${index}`, status: "draft", lock_version: 1 }));
  render(<ChildContentWorkflow outbreakId="o1" kind="resource" empty="none" items={items} documentKinds={kinds} onChanged={vi.fn(async () => {})} pageSize={5} />);
  expect(screen.getByText("Resource 5")).toBeTruthy();
  expect(screen.queryByText("Resource 6")).toBeNull();
  expect(screen.getByText("Showing 1–5 of 7")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Previous" }) as HTMLButtonElement).disabled).toBe(true);

  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText("Resource 7")).toBeTruthy();
  expect(screen.queryByText("Resource 1")).toBeNull();
  expect(screen.getByText("Page 2 of 2")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Next" }) as HTMLButtonElement).disabled).toBe(true);
});

test("workflow buttons only appear when the backend will accept them", async () => {
  render(<ChildContentWorkflow outbreakId="o1" kind="resource" empty="none" documentKinds={kinds} onChanged={vi.fn(async () => {})} currentUserId="me" parentPublished={false} items={[
    { ...base, id: "mine", title: "Mine", url: "/public/guidelines/g1", status: "pending_review", author_id: "me", lock_version: 1 },
    { ...base, id: "approved", title: "Approved", url: "/public/guidelines/g1", status: "pending_review", approved_at: "2026-09-24T10:00:00Z", lock_version: 2 },
  ]} />);
  expect(screen.getByText("You created this resource, so someone else has to approve it.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Approve" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getAllByRole("button", { name: "Approve" })).toHaveLength(1);
  expect(screen.getByText("Publish the outbreak before publishing this resource.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Publish" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.queryByRole("button", { name: "Withdraw" })).toBeNull();
});

function renderUpdates(items: Record<string, unknown>[], onChanged = vi.fn(async () => {})) {
  render(<ChildContentWorkflow outbreakId="o1" kind="update" empty="none" items={items} onChanged={onChanged} />);
  return onChanged;
}

test("editing a published update requires a reason and submits the edited fields for review", async () => {
  renderUpdates([{ id: "live", outbreak_id: "o1", title: "Cases rising", summary: "Twelve cases.", status: "published", lock_version: 2 }]);
  expect(screen.queryByRole("button", { name: "Create correction" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog");
  expect(within(dialog).getByText("Edit published update")).toBeTruthy();

  fireEvent.change(within(dialog).getByLabelText(/^Summary/), { target: { value: "Fourteen cases." } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Submit edit for review" }));
  expect(await within(dialog).findByText("A reason is required.")).toBeTruthy();
  expect(service.editPublishedUpdate).not.toHaveBeenCalled();

  fireEvent.change(within(dialog).getByLabelText(/Reason for this change/), { target: { value: "Late reports" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Submit edit for review" }));
  await waitFor(() => expect(service.editPublishedUpdate).toHaveBeenCalledWith("o1", "live", {
    lock_version: 2,
    reason: "Late reports",
    changes: { title: "Cases rising", summary: "Fourteen cases." },
  }));
});

test("a pending update edit is nested under the live update and can be discarded", async () => {
  const onChanged = renderUpdates([
    { id: "old", outbreak_id: "o1", title: "Cases (old)", status: "withdrawn", withdrawal_reason: "superseded by approved correction", lock_version: 3 },
    { id: "live", outbreak_id: "o1", title: "Cases rising", status: "published", supersedes_id: "old", lock_version: 2 },
    { id: "edit", outbreak_id: "o1", title: "Cases rising fast", status: "pending_review", supersedes_id: "live", lock_version: 1 },
  ]);
  expect(screen.queryByText("Cases (old)")).toBeNull();
  expect(screen.getByText("Edit pending review")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Edit pending change" })).toBeTruthy();

  vi.spyOn(window, "confirm").mockReturnValue(true);
  fireEvent.click(screen.getByRole("button", { name: "Discard" }));
  await waitFor(() => expect(service.deleteUpdate).toHaveBeenCalledWith("o1", "edit", 1));
  expect(onChanged).toHaveBeenCalled();
});

test("editing a draft update saves the changes in place", async () => {
  renderUpdates([{ id: "draft", outbreak_id: "o1", title: "Cases rising", summary: "", status: "draft", lock_version: 4 }]);
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/^Title/), { target: { value: "Cases rising sharply" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(service.updateUpdate).toHaveBeenCalledWith("o1", "draft", { title: "Cases rising sharply", summary: "", lock_version: 4 }));
  expect(service.editPublishedUpdate).not.toHaveBeenCalled();
});
