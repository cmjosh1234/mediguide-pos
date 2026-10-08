import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { BackendRequestError } from "@/lib/backend-client";

const service = vi.hoisted(() => ({ delete: vi.fn() }));
const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));

vi.mock("@/services/outbreaks.service", () => ({ outbreaksService: service }));
vi.mock("@/lib/toast", () => ({ showToast: toast }));

import { DeleteOutbreakDialog } from "./delete-outbreak-dialog";

const outbreak = { id: "o1", title: "Bundibugyo virus disease response", status: "contained", lock_version: 4 };

function renderDialog() {
  const onOpenChange = vi.fn(), onDeleted = vi.fn();
  render(<DeleteOutbreakDialog outbreak={outbreak} onOpenChange={onOpenChange} onDeleted={onDeleted} />);
  return { onOpenChange, onDeleted };
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

test("a live outbreak is deleted only with a reason, and linked content is reported as kept", async () => {
  service.delete.mockResolvedValue({ deleted_updates: 2, unlinked_reports: 3, unlinked_hubs: 1 });
  const { onOpenChange, onDeleted } = renderDialog();
  expect(screen.getByText(/removes it from the MediGuide app straight away/)).toBeTruthy();

  const confirm = screen.getByRole("button", { name: "Delete outbreak" }) as HTMLButtonElement;
  expect(confirm.disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "  Entered twice  " } });
  fireEvent.click(confirm);

  await waitFor(() => expect(service.delete).toHaveBeenCalledWith("o1", 4, "Entered twice"));
  expect(toast.success).toHaveBeenCalledWith("Outbreak deleted", "Kept and unlinked 3 situation reports and 1 content hub.");
  expect(onOpenChange).toHaveBeenCalledWith(false);
  expect(onDeleted).toHaveBeenCalled();
});

test("a stale list explains the conflict and keeps the dialog open", async () => {
  service.delete.mockRejectedValue(new BackendRequestError("outbreak content changed; reload and retry", 409));
  const { onDeleted } = renderDialog();
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Entered twice" } });
  fireEvent.click(screen.getByRole("button", { name: "Delete outbreak" }));

  expect((await screen.findByRole("alert")).textContent).toMatch(/refresh the list and try again/);
  expect(onDeleted).not.toHaveBeenCalled();
});
