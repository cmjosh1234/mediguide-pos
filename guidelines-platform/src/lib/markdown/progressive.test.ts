import { describe, expect, it } from "vitest";

import { splitMarkdownForProgressiveRendering } from "./progressive";

describe("progressive Markdown chunks", () => {
  it("splits chapters without cutting their content", () => {
    const chunks = splitMarkdownForProgressiveRendering(
      "Front matter\n\n## Chapter one\n\nCare one.\n\n## Chapter two\n\nCare two.",
    );

    expect(chunks.map((chunk) => chunk.content)).toEqual([
      "Front matter",
      "## Chapter one\n\nCare one.",
      "## Chapter two\n\nCare two.",
    ]);
  });

  it("keeps duplicate heading ids unique across separately rendered chunks", () => {
    const chunks = splitMarkdownForProgressiveRendering(
      "## Assessment\n\n### Treatment\n\nFirst.\n\n## Follow-up\n\n### Treatment\n\nSecond.",
    );

    expect(chunks.flatMap((chunk) => chunk.headingIds)).toEqual([
      "assessment",
      "treatment",
      "follow-up",
      "treatment-1",
    ]);
  });
});
