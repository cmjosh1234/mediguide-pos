import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  GuidelineReviewIssuesPopover,
  groupReviewIssues,
} from "./guideline-review-issues";

const pending = (index: number) => ({
  code: "unreviewed_high_risk_block",
  message: `The table "Table ${index}" block requires publisher review.`,
  remediation: "Review this high-risk clinical block individually before publication.",
  block_id: `block-${index}`,
});

describe("groupReviewIssues", () => {
  it("summarises the review queue and lists real defects first", () => {
    const groups = groupReviewIssues([
      pending(1),
      pending(2),
      pending(3),
      { code: "invalid_block_payload", message: "Table has no rows.", block_id: "block-9" },
    ]);

    expect(groups.map((group) => group.code)).toEqual([
      "invalid_block_payload",
      "unreviewed_high_risk_block",
    ]);
    expect(groups[1].title).toBe("3 high-risk blocks awaiting publisher review");
    expect(groups[1].reviewQueue).toBe(true);
    expect(groups[1].issues).toHaveLength(3);
    expect(groups[0].title).toBe("Table has no rows.");
  });

  it("collapses repeated defects into one row", () => {
    const [group] = groupReviewIssues([
      { code: "broken_block_section", message: "The block references a missing section." },
      { code: "broken_block_section", message: "The block references a missing section." },
    ]);
    expect(group.title).toBe("The block references a missing section. (+1 more)");
  });
});

describe("GuidelineReviewIssuesPopover", () => {
  afterEach(cleanup);

  it("shows ready state without blockers", () => {
    render(
      <GuidelineReviewIssuesPopover
        errors={[]}
        warnings={[]}
        extractionWarnings={[]}
        onFocusIssue={vi.fn()}
        onStartReview={vi.fn()}
      />,
    );
    expect(screen.getByText("Ready to publish")).toBeInTheDocument();
  });

  it("starts the review queue from the grouped summary", async () => {
    const onStartReview = vi.fn();
    render(
      <GuidelineReviewIssuesPopover
        errors={[pending(1), pending(2)]}
        warnings={[]}
        extractionWarnings={[]}
        onFocusIssue={vi.fn()}
        onStartReview={onStartReview}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: /1 blocker/ }));
    expect(
      screen.getByText("2 high-risk blocks awaiting publisher review"),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Table 1/)).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Start reviewing" }));
    expect(onStartReview).toHaveBeenCalledOnce();
  });
});
