import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import type { PublicGuideline } from "../../../api/public-guidelines";
import { LinkedDocumentReader } from "./LinkedDocumentReader";

const guideline: PublicGuideline = {
  id: "343fd1e0-ef83-4034-91b6-e3f443a537f9",
  slug: "ucg",
  title: "UCG",
  description: "",
  country: "Uganda",
  source_org: "MOH",
  program_area: "GENERAL",
  language: "en",
  publication_date: "2026-10-06",
  review_date: "2026-10-08",
  version: "2026.01",
  last_updated: "2026-10-06T22:33:40Z",
  document_kind: { slug: "link", name: "Link", publish_as_uploaded: false, publish_as_link: true },
  external_url: "https://who.int",
};

const render = (value: PublicGuideline) =>
  renderToStaticMarkup(
    <MemoryRouter>
      <LinkedDocumentReader guideline={value} />
    </MemoryRouter>,
  );

describe("LinkedDocumentReader", () => {
  it("says where the link goes and opens it in a new tab", () => {
    const html = render(guideline);
    expect(html).toContain("UCG");
    expect(html).toContain("External website");
    expect(html).toContain("who.int");
    expect(html).toContain('href="https://who.int"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain("Open website");
  });

  it("offers nothing to open when the link isn't https", () => {
    const html = render({ ...guideline, external_url: "javascript:alert(1)" });
    expect(html).toContain("This link is unavailable.");
    expect(html).not.toContain("Open website");
  });
});
