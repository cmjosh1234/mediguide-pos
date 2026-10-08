import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { BackendRequestError } from "@/lib/backend-client";

const outbreak = {
  id: "o1",
  title: "Measles response",
  status: "draft",
  disease_id: "d1",
  visual_tone: "warning",
  lock_version: 1,
  start_date: "2026-06-12T00:00:00Z",
  effective_at: "2026-06-11T12:00:00Z",
  metrics: [],
};

const service = vi.hoisted(() => ({
  get: vi.fn(),
  listUpdates: vi.fn(async () => ({ items: [] })),
  listResources: vi.fn(async () => ({ items: [] })),
  audit: vi.fn(async () => ({ items: [] })),
  listPublishedGuidelines: vi.fn(async () => ({ items: [] })),
  transition: vi.fn(),
  update: vi.fn(),
}));

vi.mock("@/services/outbreaks.service", () => ({ outbreaksService: service }));
vi.mock("@/services/situation-reports.service", () => ({ situationReportsService: { list: async () => ({ items: [] }) } }));
vi.mock("@/services/content-hubs.service", () => ({
  contentHubService: {},
  diseaseService: { list: async () => ({ items: [{ id: "d1", name: "Measles" }] }) },
}));
vi.mock("@/services/document-kinds.service", () => ({ documentKindService: { list: async () => [] } }));
vi.mock("@/services/health-facilities.service", () => ({ healthFacilitiesService: { regions: async () => [], districts: async () => [] } }));
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));

import { OutbreakEditor } from "./outbreak-editor";

beforeEach(() => {
  vi.clearAllMocks();
  service.get.mockResolvedValue(outbreak);
  Element.prototype.scrollIntoView = vi.fn();
});
afterEach(cleanup);

const dateOrder = "Effective at can't be earlier than the start date.";

test("fields named by a failed workflow step are highlighted until they are edited", async () => {
  service.transition.mockRejectedValue(
    new BackendRequestError(dateOrder, 400, undefined, {
      fields: [
        { field: "effective_at", message: dateOrder },
        { field: "start_date", message: dateOrder },
      ],
    }),
  );
  render(<OutbreakEditor id="o1" />);
  const effective = await screen.findByLabelText(/^Effective at/);
  const start = screen.getByLabelText(/^Start date/);
  // The Updates card has its own Title field.
  const title = document.getElementById("outbreak-title")!;

  fireEvent.click(screen.getByRole("button", { name: "Submit for review" }));
  await waitFor(() => expect(effective.getAttribute("aria-invalid")).toBe("true"));
  expect(start.getAttribute("aria-invalid")).toBe("true");
  expect(title.getAttribute("aria-invalid")).toBe("false");
  expect(screen.getAllByText(dateOrder)).toHaveLength(2);
  await waitFor(() => expect(document.activeElement).toBe(effective));

  // Fixing either date clears the error on both.
  fireEvent.change(start, { target: { value: "2026-06-10T00:00" } });
  expect(effective.getAttribute("aria-invalid")).toBe("false");
  expect(start.getAttribute("aria-invalid")).toBe("false");
  expect(screen.queryByText(dateOrder)).toBeNull();
});

test("unsaved changes block the workflow until the draft is saved", async () => {
  service.update.mockImplementation(async (_id: string, payload: Record<string, unknown>) => ({
    ...outbreak,
    effective_at: payload.effective_at,
    lock_version: 2,
  }));
  render(<OutbreakEditor id="o1" />);
  const effective = (await screen.findByLabelText(/^Effective at/)) as HTMLInputElement;
  const loaded = effective.value;
  const submit = screen.getByRole("button", { name: "Submit for review" }) as HTMLButtonElement;
  expect(submit.disabled).toBe(false);
  expect(screen.queryByText(/save the draft first/)).toBeNull();

  fireEvent.change(effective, { target: { value: "2026-06-14T12:00" } });
  expect(screen.getByText(/save the draft first/)).toBeTruthy();
  expect(submit.disabled).toBe(true);

  // Putting the saved value back means there is nothing to save.
  fireEvent.change(effective, { target: { value: loaded } });
  expect(screen.queryByText(/save the draft first/)).toBeNull();
  fireEvent.change(effective, { target: { value: "2026-06-14T12:00" } });

  fireEvent.click(screen.getAllByRole("button", { name: "Save draft" })[0]);
  await waitFor(() => expect(service.update).toHaveBeenCalled());
  await waitFor(() => expect(screen.queryByText(/save the draft first/)).toBeNull());
  expect(submit.disabled).toBe(false);
  expect(service.transition).not.toHaveBeenCalled();
});

test("a new outbreak is saved from the end of the form, not the page header", async () => {
  render(<OutbreakEditor />);
  await screen.findByText("New outbreak");
  const header = screen.getByRole("link", { name: "Back to list" }).parentElement!;
  expect(within(header).queryByRole("button", { name: "Save draft" })).toBeNull();
  expect(screen.getAllByRole("button", { name: "Save draft" })).toHaveLength(1);
});

test("a correction is saved from its workflow card, not the page header", async () => {
  const live = { ...outbreak, id: "o0", status: "active", published_at: "2026-06-12T00:00:00Z" };
  service.get.mockImplementation(async (id: string) => (id === "o0" ? live : { ...outbreak, id: "c1", supersedes_id: "o0" }));
  render(<OutbreakEditor id="c1" />);
  const effective = await screen.findByLabelText(/^Effective at/);
  const header = screen.getByRole("link", { name: "Back to list" }).parentElement!;
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();

  fireEvent.change(effective, { target: { value: "2026-06-14T12:00" } });
  expect(within(header).queryByRole("button", { name: "Save draft" })).toBeNull();
  const card = screen.getByText("Correction workflow").closest("[data-slot=card]") as HTMLElement;
  expect(within(card).getByRole("button", { name: "Save draft" })).toBeTruthy();
});

test("each missing field keeps its own highlight", async () => {
  service.transition.mockRejectedValue(
    new BackendRequestError("Before publishing, fill in and save: Geographic coverage, Source reference.", 400, undefined, {
      fields: [
        { field: "geographic_area", message: "Geographic coverage is needed to publish." },
        { field: "source_reference", message: "Source reference is needed to publish." },
      ],
    }),
  );
  render(<OutbreakEditor id="o1" />);
  const area = await screen.findByLabelText(/^Geographic coverage/);
  const reference = screen.getByLabelText(/^Source reference/);

  fireEvent.click(screen.getByRole("button", { name: "Submit for review" }));
  expect(await screen.findByText("Geographic coverage is needed to publish.")).toBeTruthy();
  fireEvent.change(area, { target: { value: "Gulu, Uganda" } });
  expect(screen.queryByText("Geographic coverage is needed to publish.")).toBeNull();
  expect(reference.getAttribute("aria-invalid")).toBe("true");
});
