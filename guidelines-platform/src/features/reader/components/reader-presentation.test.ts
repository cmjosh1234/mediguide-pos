import { describe, expect, it } from "vitest";

import type {
  PublicGuideline,
  PublicGuidelineBlock,
  PublicGuidelineSection,
} from "../../../api/public-guidelines";
import {
  readerBlocks,
  readerNavigationSections,
  readerSections,
  removeLeadingDocumentTitle,
} from "./reader-presentation";

const guideline = {
  id: "diabetes",
  title: "Integrated Diabetes Management Guideline 2026",
} as PublicGuideline;

const title: PublicGuidelineSection = {
  id: "title",
  title: "Integrated Diabetes Management Guideline 2026",
  slug: "integrated-diabetes-management-guideline-2026",
  level: 1,
  sort_order: 0,
};

const chapter: PublicGuidelineSection = {
  id: "introduction",
  parent_id: "title",
  title: "Introduction and Preface",
  slug: "introduction-and-preface",
  level: 2,
  sort_order: 1,
};

const titleHeading: PublicGuidelineBlock = {
  id: "title-heading",
  section_id: "title",
  type: "heading",
  sort_order: 0,
  content: {
    text: "# Integrated Diabetes Management Guideline 2026",
    level: 1,
  },
};

describe("guideline reader presentation", () => {
  it("keeps extracted headings in source data but omits the duplicate display block", () => {
    const paragraph: PublicGuidelineBlock = {
      id: "body",
      section_id: "title",
      type: "paragraph",
      sort_order: 1,
      content: { text: "Reviewed introduction." },
    };

    expect(readerBlocks(title, [paragraph, titleHeading]).map((item) => item.id)).toEqual([
      "body",
    ]);
  });

  it("removes an empty H1 wrapper and promotes its chapters in navigation", () => {
    expect(readerSections(guideline, [title, chapter], [titleHeading])).toEqual([
      chapter,
    ]);
    expect(readerNavigationSections(guideline, [title, chapter])).toEqual([
      chapter,
    ]);
  });

  it("removes the source H1 from book content regardless of harmless title formatting", () => {
    expect(
      removeLeadingDocumentTitle(
        "# **Integrated Diabetes Management Guideline — 2026**\n\n## Introduction\n\nBody",
      ),
    ).toBe("## Introduction\n\nBody");
  });
});
