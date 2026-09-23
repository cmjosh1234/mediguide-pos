import { afterEach, describe, expect, it, vi } from "vitest";

import {
  assignableDocumentKinds,
  documentKindService,
  normalizeDocumentKind,
} from "./document-kinds.service";

const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify({ success: true, data }), {
    status,
    headers: { "Content-Type": "application/json" },
  });

describe("document kind service", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("lists kinds in display order and normalizes missing fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json({
        items: [
          { id: "k1", name: "Guideline", slug: "guideline", sort_order: 10, status: "active", guideline_document_count: 3 },
          { id: "k2", name: "Contact Tracing Guide", slug: "contact_tracing_guide", outbreak_document_count: 1 },
        ],
        page: 1,
        per_page: 100,
        total_items: 2,
        total_pages: 1,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const kinds = await documentKindService.list("active");

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain("/api/v2/document-kinds");
    expect(url).toContain("sort=sort_order");
    expect(url).toContain("status=active");
    expect(kinds).toEqual([
      { id: "k1", name: "Guideline", slug: "guideline", description: "", sort_order: 10, status: "active", publish_as_uploaded: false, guideline_document_count: 3, outbreak_document_count: 0 },
      { id: "k2", name: "Contact Tracing Guide", slug: "contact_tracing_guide", description: "", sort_order: 0, status: "active", publish_as_uploaded: false, guideline_document_count: 0, outbreak_document_count: 1 },
    ]);
  });

  it("sends trimmed payloads and uses the kind id for update and delete", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json({ id: "k2", name: "Form", slug: "form", status: "active" }, 201))
      .mockResolvedValueOnce(json({ id: "k2", name: "Forms", slug: "form", status: "inactive" }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await documentKindService.create({ name: "  Form  ", description: " Reporting tools " });
    const updated = await documentKindService.update("k2", { name: "Forms", slug: "form", status: "inactive", sort_order: 20, publish_as_uploaded: true });
    await documentKindService.delete("k2");

    const [createUrl, createInit] = fetchMock.mock.calls[0];
    expect(String(createUrl)).toMatch(/\/api\/v2\/document-kinds$/);
    expect(createInit.method).toBe("POST");
    expect(JSON.parse(createInit.body)).toEqual({ name: "Form", slug: "", description: "Reporting tools", sort_order: 0, status: "active", publish_as_uploaded: false });

    const [updateUrl, updateInit] = fetchMock.mock.calls[1];
    expect(String(updateUrl)).toContain("/api/v2/document-kinds/k2");
    expect(updateInit.method).toBe("PATCH");
    expect(JSON.parse(updateInit.body).publish_as_uploaded).toBe(true);
    expect(updated.status).toBe("inactive");

    const [deleteUrl, deleteInit] = fetchMock.mock.calls[2];
    expect(String(deleteUrl)).toContain("/api/v2/document-kinds/k2");
    expect(deleteInit.method).toBe("DELETE");
  });

  it("offers active kinds plus the document's current inactive kind", () => {
    const kinds = [
      normalizeDocumentKind({ id: "a", slug: "sop", status: "active" }),
      normalizeDocumentKind({ id: "b", slug: "legacy", status: "inactive" }),
      normalizeDocumentKind({ id: "c", slug: "retired", status: "inactive" }),
    ];
    expect(assignableDocumentKinds(kinds).map((kind) => kind.slug)).toEqual(["sop"]);
    expect(assignableDocumentKinds(kinds, "legacy").map((kind) => kind.slug)).toEqual(["sop", "legacy"]);
    expect(assignableDocumentKinds(kinds, "c").map((kind) => kind.slug)).toEqual(["sop", "retired"]);
  });
});
