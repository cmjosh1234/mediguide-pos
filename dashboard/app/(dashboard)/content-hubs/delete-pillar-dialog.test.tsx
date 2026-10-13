import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { BackendRequestError } from "@/lib/backend-client";
import type { ContentPillar } from "@/services/content-hubs.service";

const service = vi.hoisted(() => ({ deletePillar: vi.fn() }));
const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));

vi.mock("@/services/content-hubs.service", () => ({ contentHubService: service }));
vi.mock("@/lib/toast", () => ({ showToast: toast }));

import { DeletePillarDialog } from "./delete-pillar-dialog";

const pillar = (id: string, name: string, parent_id?: string, items = 0) =>
  ({ id, name, parent_id, hub_id: "h1", slug: id, sort_order: 0, status: "active", lock_version: 3, items: Array.from({ length: items }, (_, index) => ({ id: `${id}-${index}` })) }) as unknown as ContentPillar;
const treatment = pillar("p1", "Treatment", undefined, 1);
const pillars = [treatment, pillar("p2", "Severe malaria", "p1", 2), pillar("p3", "Overview")];

function renderDialog() {
  const onOpenChange = vi.fn(), onDeleted = vi.fn();
  render(<DeletePillarDialog hubId="h1" live pillar={treatment} pillars={pillars} onOpenChange={onOpenChange} onDeleted={onDeleted} />);
  return { onOpenChange, onDeleted };
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("a pillar is deleted only with a reason, and what goes with it is spelled out", async () => {
  service.deletePillar.mockResolvedValue(undefined);
  const { onOpenChange, onDeleted } = renderDialog();
  expect(screen.getByText(/removes it from the MediGuide app and website straight away/)).toBeTruthy();
  expect(screen.getByText(/1 sub-pillar and 3 resource assignments \(Severe malaria\)/)).toBeTruthy();

  const confirm = screen.getByRole("button", { name: "Delete pillar" }) as HTMLButtonElement;
  expect(confirm.disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "  Merged into Clinical Management  " } });
  fireEvent.click(confirm);

  await waitFor(() => expect(service.deletePillar).toHaveBeenCalledWith("h1", "p1", 3, "Merged into Clinical Management"));
  expect(onOpenChange).toHaveBeenCalledWith(false);
  expect(onDeleted).toHaveBeenCalled();
});

test("a stale page explains the conflict and keeps the dialog open", async () => {
  service.deletePillar.mockRejectedValue(new BackendRequestError("content hub was modified by another request", 409));
  const { onDeleted } = renderDialog();
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Duplicate" } });
  fireEvent.click(screen.getByRole("button", { name: "Delete pillar" }));

  expect((await screen.findByRole("alert")).textContent).toMatch(/refresh the page and try again/);
  expect(onDeleted).not.toHaveBeenCalled();
});
