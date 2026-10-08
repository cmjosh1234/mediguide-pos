import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import type { PublicSituationReport } from "../../api/public-guidelines";
import { SituationReportView } from "./SituationReportPage";

const documentId = "11111111-1111-4111-8111-111111111111";

const report: PublicSituationReport = {
  id: "r1",
  title: "Bundibugyo virus disease weekly external situation report 11",
  summary: "Continued transmission in the Democratic Republic of the Congo.",
  source_organization: "WHO Regional Office for Africa",
  source_reference: "WHO weekly external situation report 11",
  publication_date: "2026-07-26T12:00:00Z",
  metrics: [
    { key: "contacts", label: "Contacts followed up", value: "836", unit: "contacts", sort_order: 2 },
    { key: "deaths", label: "Deaths in Uganda", value: "2", unit: "deaths", sort_order: 1 },
  ],
  key_highlights: ["Regional cross-border spread risk remained high"],
  attachments: [
    { id: "a1", title: "Full report", document_kind: "situation_report_attachment", url: `/public/guidelines/${documentId}`, sort_order: 1 },
  ],
  report_asset_url: `/api/public/guidelines/${documentId}/original/download`,
};

const render = (value: PublicSituationReport) =>
  renderToStaticMarkup(
    <MemoryRouter>
      <SituationReportView report={value} />
    </MemoryRouter>,
  );

describe("SituationReportView", () => {
  it("shows the report's figures, highlights and attached documents", () => {
    const html = render(report);
    expect(html).toContain("Bundibugyo virus disease weekly external situation report 11");
    expect(html).toContain("WHO Regional Office for Africa");
    expect(html).toContain("Key figures");
    // Figures follow their sort order.
    expect(html.indexOf("Deaths in Uganda")).toBeLessThan(html.indexOf("Contacts followed up"));
    expect(html).toContain("Regional cross-border spread risk remained high");
    expect(html).toContain("Report documents");
    expect(html).toContain(`href="/guidelines/${documentId}"`);
    // With attachments, the single-PDF link isn't offered as well.
    expect(html).not.toContain("Download full report");
  });

  it("offers the PDF of a report from before attachments", () => {
    const html = render({ ...report, attachments: [], report_asset_url: "/api/public/situation-reports/r1/asset" });
    expect(html).toContain("Download full report");
    expect(html).toContain("/api/public/situation-reports/r1/asset");
    expect(html).not.toContain("Report documents");
  });
});
