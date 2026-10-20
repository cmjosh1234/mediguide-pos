import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import CalculatorsPage from "./page";
import { BackendRequestError } from "@/lib/backend-client";
import { SupportTicketsService } from "@/services/support-tickets.service";
import { showToast } from "@/lib/toast";
import { toast } from "sonner";

vi.mock("next/navigation", () => ({ useRouter: () => ({ replace: vi.fn(), push: vi.fn() }) }));
vi.mock("@/lib/permission-context", () => ({
  usePermissionContext: () => ({ hasPermission: () => true, loading: false }),
}));
vi.mock("@/lib/toast", () => ({ showToast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/lib/backend-client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/backend-client")>()),
  getCurrentUser: () => ({ id: "u1" }),
}));
vi.mock("@/services/support-tickets.service", () => ({
  SupportTicketsService: { createTicket: vi.fn() },
}));
vi.mock("@/hooks/use-tool-categories", () => ({
  useToolCategories: () => ({ activeCategories: [{ id: "c1", name: "Neurology" }] }),
}));
vi.mock("@/services/calculator.service", () => ({
  calculatorService: {
    list: vi.fn(async () => ({
      items: [
        { id: "t1", name: "APGAR Score Calculator", type: "calculator" },
        { id: "t2", name: "BMI Calculator", type: "calculator" },
      ],
    })),
  },
}));
vi.mock("@/services/clinical-tool.service", () => ({
  clinicalToolService: { definition: vi.fn(() => new Promise(() => {})) },
}));

const scrollIntoView = vi.fn();
beforeEach(() => {
  scrollIntoView.mockClear();
  Element.prototype.scrollIntoView = scrollIntoView;
});
afterEach(cleanup);

describe("Clinical tools runner", () => {
  it("shows the summary as one compact strip", async () => {
    render(<CalculatorsPage />);
    expect(await screen.findByText("Published tools")).toBeInTheDocument();
    expect(screen.getByText("JSON schema v1")).toBeInTheDocument();
  });

  it("marks the chosen tool and scrolls its panel into view", async () => {
    render(<CalculatorsPage />);
    const bmi = await screen.findByRole("button", { name: /BMI Calculator/ });
    await waitFor(() => expect(screen.getByRole("button", { name: /APGAR/ })).toHaveAttribute("aria-pressed", "true"));
    expect(scrollIntoView).not.toHaveBeenCalled();

    fireEvent.click(bmi);

    expect(bmi).toHaveAttribute("aria-pressed", "true");
    expect(scrollIntoView).toHaveBeenCalledWith(expect.objectContaining({ block: "start" }));
  });

  async function fillAndSubmit() {
    render(<CalculatorsPage />);
    fireEvent.click(await screen.findByRole("button", { name: /Request Tool/ }));
    expect(screen.getByText(/sent to the support team as a ticket/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Name *"), { target: { value: "Sepsis screen" } });
    fireEvent.change(screen.getByLabelText("Description *"), { target: { value: "qSOFA-based screening" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit request" }));
  }

  it("logs the request as a support ticket and offers a link to it", async () => {
    vi.mocked(SupportTicketsService.createTicket).mockResolvedValue({ id: "ticket-9" } as never);
    await fillAndSubmit();
    await waitFor(() => expect(SupportTicketsService.createTicket).toHaveBeenCalled());
    expect(vi.mocked(SupportTicketsService.createTicket).mock.calls[0][0]).toMatchObject({
      subject: "Clinical Tool Request: Sepsis screen",
      category: "Clinical Tool Request",
      priority: "normal",
    });
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith(
      "Request logged as a support ticket",
      expect.objectContaining({ action: expect.objectContaining({ label: "View ticket" }) }),
    ));
    await waitFor(() => expect(screen.queryByText(/sent to the support team as a ticket/)).toBeNull());
  });

  it("explains the hourly limit when it is reached", async () => {
    vi.mocked(SupportTicketsService.createTicket).mockRejectedValue(new BackendRequestError("rate limit exceeded", 429));
    await fillAndSubmit();
    await waitFor(() => expect(showToast.error).toHaveBeenCalledWith(
      "Submission failed",
      "You can send up to 5 requests an hour. Try again later.",
    ));
  });
});
