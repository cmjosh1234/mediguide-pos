import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import type { PublicGuideline } from "../../api/public-guidelines";
import { publicApiBaseUrl } from "../../config";
import { BackendGuidelineCard } from "./LandingPage";

const guideline: PublicGuideline = {
  id: "guideline-id",
  slug: "diabetes-care",
  title: "Diabetes Care",
  description: "Reviewed clinical guidance.",
  country: "Uganda",
  source_org: "Ministry of Health",
  program_area: "Non-communicable diseases",
  language: "en",
  publication_date: "2026-01-01",
  review_date: "2027-01-01",
  version: "2026.1",
  last_updated: "2026-01-01T00:00:00Z",
};

function renderCard(value: PublicGuideline) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <BackendGuidelineCard guideline={value} />
    </MemoryRouter>,
  );
}

describe("BackendGuidelineCard", () => {
  it("places right-aligned read and download actions on downloadable publications", () => {
    const html = renderCard({ ...guideline, has_original_document: true });

    expect(html).toContain('class="card-actions"');
    expect(html).toContain('href="/guidelines/guideline-id"');
    expect(html).toContain("Read guideline");
    expect(html).toContain("Download");
    expect(html).toContain(
      `href="${publicApiBaseUrl}/api/public/guidelines/guideline-id/original/download"`,
    );
  });

  it("does not offer a broken download when no original document exists", () => {
    const html = renderCard(guideline);

    expect(html).toContain("Read guideline");
    expect(html).not.toContain("Download");
    expect(html).not.toContain("original/download");
  });
});
