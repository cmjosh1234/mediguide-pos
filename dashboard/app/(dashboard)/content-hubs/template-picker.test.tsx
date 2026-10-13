import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { BackendRequestError } from "@/lib/backend-client";
import type { ContentHubWorkspace, ContentPillar, HubTemplate } from "@/services/content-hubs.service";

const service = vi.hoisted(() => ({ applyTemplate: vi.fn() }));
const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));

vi.mock("@/services/content-hubs.service", () => ({ contentHubService: service }));
vi.mock("@/lib/toast", () => ({ showToast: toast }));

import { ReplaceTemplateDialog, TemplatePicker } from "./template-picker";

const pillar = (id: string, name: string, items = 0) =>
  ({ id, name, hub_id: "h1", slug: id, sort_order: 0, status: "active", lock_version: 1, items: Array.from({ length: items }, (_, index) => ({ id: `${id}-${index}` })) }) as unknown as ContentPillar;
const surveillance = { id: "t2", name: "Surveillance knowledge", slug: "surveillance", pillars: [{ id: "d1", name: "Surveillance", slug: "surveillance", sort_order: 10 }, { id: "d2", name: "Reporting", slug: "reporting", sort_order: 20 }] } as HubTemplate;
const workspace = (status: string, pillars: ContentPillar[]) => ({ hub: { id: "h1", status, lock_version: 7 }, pillars }) as unknown as ContentHubWorkspace;

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("each hub state explains what the template button will do", () => {
  render(<TemplatePicker workspace={workspace("draft", [])} templates={[surveillance]} onApplied={vi.fn()} />);
  expect(screen.getByText(/Copies the template's pillars into this hub/)).toBeTruthy();
  expect((screen.getByRole("button", { name: "Apply template" }) as HTMLButtonElement).disabled).toBe(true);
  cleanup();

  render(<TemplatePicker workspace={workspace("draft", [pillar("p1", "Overview")])} templates={[surveillance]} onApplied={vi.fn()} />);
  expect(screen.getByText(/Replacing the template deletes them and their resource assignments first/)).toBeTruthy();
  expect(screen.getByRole("button", { name: "Replace template" })).toBeTruthy();
  cleanup();

  render(<TemplatePicker workspace={workspace("active", [pillar("p1", "Overview")])} templates={[surveillance]} onApplied={vi.fn()} />);
  expect(screen.getByText(/This hub is published, so change its pillars below instead/)).toBeTruthy();
  expect((screen.getByRole("button", { name: "Apply template" }) as HTMLButtonElement).disabled).toBe(true);
});

test("replacing spells out what is deleted and added, and needs a reason", async () => {
  service.applyTemplate.mockResolvedValue([]);
  const onOpenChange = vi.fn(), onReplaced = vi.fn();
  render(<ReplaceTemplateDialog hubId="h1" lockVersion={7} template={surveillance} pillars={[pillar("p1", "Overview", 2), pillar("p2", "Diagnosis", 1)]} onOpenChange={onOpenChange} onReplaced={onReplaced} />);
  expect(screen.getByText(/2 pillars and 3 resource assignments \(Overview, Diagnosis\)/)).toBeTruthy();
  expect(screen.getByText(/2 pillars from the template \(Surveillance, Reporting\)/)).toBeTruthy();

  const confirm = screen.getByRole("button", { name: "Replace template" }) as HTMLButtonElement;
  expect(confirm.disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "  Surveillance layout  " } });
  fireEvent.click(confirm);

  await waitFor(() => expect(service.applyTemplate).toHaveBeenCalledWith("h1", "t2", 7, "Surveillance layout"));
  expect(onOpenChange).toHaveBeenCalledWith(false);
  await waitFor(() => expect(onReplaced).toHaveBeenCalled());
});

test("a stale page explains the conflict and keeps the dialog open", async () => {
  service.applyTemplate.mockRejectedValue(new BackendRequestError("content hub was modified by another request", 409));
  const onReplaced = vi.fn();
  render(<ReplaceTemplateDialog hubId="h1" lockVersion={7} template={surveillance} pillars={[pillar("p1", "Overview")]} onOpenChange={vi.fn()} onReplaced={onReplaced} />);
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Surveillance layout" } });
  fireEvent.click(screen.getByRole("button", { name: "Replace template" }));

  expect((await screen.findByRole("alert")).textContent).toMatch(/reload the page and try again/);
  expect(onReplaced).not.toHaveBeenCalled();
});
