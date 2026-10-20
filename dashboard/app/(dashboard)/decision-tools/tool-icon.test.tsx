import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ToolIcon } from "./tool-icon";

const fallback = <span data-testid="fallback" />;

describe("ToolIcon", () => {
  it("never renders a Lucide icon name as text", () => {
    const { container } = render(<ToolIcon icon="heart-pulse" fallback={fallback} />);
    expect(container.textContent).not.toContain("heart-pulse");
  });

  it("renders emoji icons as text", () => {
    render(<ToolIcon icon="🧮" fallback={fallback} />);
    expect(screen.getByText("🧮")).toBeInTheDocument();
  });

  it("falls back for empty or unknown identifiers", () => {
    const { rerender } = render(<ToolIcon icon="not-a-real-icon" fallback={fallback} />);
    expect(screen.getByTestId("fallback")).toBeInTheDocument();
    expect(screen.queryByText("not-a-real-icon")).toBeNull();
    rerender(<ToolIcon fallback={fallback} />);
    expect(screen.getByTestId("fallback")).toBeInTheDocument();
  });
});
