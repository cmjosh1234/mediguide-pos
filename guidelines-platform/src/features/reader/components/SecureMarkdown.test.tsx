import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { SecureMarkdown } from "./SecureMarkdown";

describe("SecureMarkdown", () => {
  it("renders GFM tables and task lists", () => {
    const html = renderToStaticMarkup(
      <SecureMarkdown content={"- [x] Reviewed\n\n| Drug | Dose |\n| --- | --- |\n| ACT | 1 |"} />,
    );
    expect(html).toContain("<table>");
    expect(html).toContain('type="checkbox"');
    expect(html).toContain("Reviewed");
  });

  it("does not enable raw HTML or unsafe URLs", () => {
    const html = renderToStaticMarkup(
      <SecureMarkdown content={'<script>alert(1)</script>\n\n[Unsafe](javascript:alert(1))\n\n![Bad](data:text/html,bad)'} />,
    );
    expect(html).not.toContain("<script");
    expect(html).not.toContain("javascript:");
    expect(html).not.toContain("data:text/html");
    expect(html).toContain("Image unavailable");
  });

  it("adds safe attributes to external links and stable heading IDs", () => {
    const html = renderToStaticMarkup(
      <SecureMarkdown content={"## Care plan\n\n[WHO](https://www.who.int)\n\n## Care plan"} />,
    );
    expect(html).toContain('id="care-plan"');
    expect(html).toContain('id="care-plan-1"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain('rel="noopener noreferrer"');
  });

  it("uses document-wide heading ids for a progressive chunk", () => {
    const html = renderToStaticMarkup(
      <SecureMarkdown
        content={"## Treatment\n\nReviewed guidance."}
        headingIds={["treatment-4"]}
      />,
    );

    expect(html).toContain('id="treatment-4"');
    expect(html).toContain('href="#treatment-4"');
  });
});
