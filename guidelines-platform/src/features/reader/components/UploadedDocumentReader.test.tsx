import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import type { PublicGuideline, PublicGuidelineAssetLink } from "../../../api/public-guidelines";
import { UploadedDocumentView, type UploadedFileState } from "./UploadedDocumentReader";

const guideline: PublicGuideline = {
  id: "g1",
  slug: "hmis-105-outpatient-register",
  title: "HMIS 105 Outpatient Register",
  description: "Monthly outpatient summary form.",
  country: "Uganda",
  source_org: "Ministry of Health",
  program_area: "",
  language: "en",
  publication_date: "2024-07-01",
  review_date: "",
  version: "2024",
  last_updated: "2024-07-01T00:00:00Z",
  document_kind: { slug: "form", name: "Form", publish_as_uploaded: true },
};

const asset = (mime_type: string, original_filename: string): PublicGuidelineAssetLink => ({
  type: "original_pdf",
  mime_type,
  original_filename,
  url: "/api/public/guidelines/g1/original/download",
  expires_at: "2024-07-01T00:10:00Z",
});

const render = (file: UploadedFileState) =>
  renderToStaticMarkup(
    <MemoryRouter>
      <UploadedDocumentView guideline={guideline} file={file} />
    </MemoryRouter>,
  );

describe("UploadedDocumentView", () => {
  it("shows the document details while the file loads", () => {
    const html = render({ status: "loading" });
    expect(html).toContain("HMIS 105 Outpatient Register");
    expect(html).toContain(">Form<");
    expect(html).toContain("Ministry of Health · Version 2024 · 2024-07-01");
    expect(html).toContain("Loading the document…");
    expect(html).not.toContain("<iframe");
  });

  it("embeds a PDF exactly as uploaded with a download link", () => {
    const html = render({ status: "ready", url: "blob:form", asset: asset("application/pdf", "hmis-105.pdf") });
    expect(html).toContain('<iframe class="uploaded-document-frame" title="HMIS 105 Outpatient Register (PDF)" src="blob:form"');
    expect(html).toContain('download="hmis-105.pdf"');
    expect(html).toContain("Download PDF");
  });

  it("offers Word documents as a download only", () => {
    const html = render({
      status: "ready",
      url: "blob:form",
      asset: asset("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "hmis-105.docx"),
    });
    expect(html).not.toContain("<iframe");
    expect(html).toContain("Download Word document");
    expect(html).toContain("This document is a Word file.");
  });

  it("reports a file that cannot be loaded", () => {
    const html = render({ status: "error" });
    expect(html).toContain('role="alert"');
    expect(html).not.toContain("Download");
  });
});
