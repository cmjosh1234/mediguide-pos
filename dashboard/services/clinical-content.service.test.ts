import { afterEach, describe, expect, it, vi } from "vitest";

import { calculatorService } from "./calculator.service";
import { drugReferenceService, drugService } from "./drug.service";

const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify({ success: true, data }), {
    status,
    headers: { "Content-Type": "application/json" },
  });

describe("typed calculator and drug services", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("serializes calculator list filters as typed query parameters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json({
        items: [],
        page: 1,
        per_page: 20,
        total_items: 0,
        total_pages: 0,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await calculatorService.listTable(
      { page: 1, perPage: 20, search: "triage", filters: [] },
      "decision_tool",
    );

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain("/api/v2/calculators");
    expect(url).toContain("search=triage");
    expect(url).toContain("type=decision_tool");
    expect(url).not.toContain("filter=");
    expect(url).not.toContain("expand=");
  });

  it("maps calculator table type and status filters to backend query parameters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json({
        items: [],
        page: 1,
        per_page: 20,
        total_items: 0,
        total_pages: 0,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await calculatorService.listTable({
      page: 1,
      perPage: 20,
      search: "",
      filters: [
        { id: "1", field: "type", condition: "equals", value: "checklist" },
        { id: "2", field: "status", condition: "not_equals", value: "archived" },
      ],
    });

    const url = new URL(String(fetchMock.mock.calls[0][0]), "http://localhost");
    expect(url.searchParams.get("type")).toBe("checklist");
    expect(url.searchParams.get("status")).toBe("active,draft");
  });

  it("maps calculator category and risk filters to backend query parameters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json({ items: [], page: 1, per_page: 20, total_items: 0, total_pages: 0 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await calculatorService.listTable({
      page: 1,
      perPage: 20,
      search: "",
      filters: [
        { id: "1", field: "category_id", condition: "equals", value: "cat-1" },
        { id: "2", field: "risk_level", condition: "not_equals", value: "standard" },
      ],
    });

    const url = new URL(String(fetchMock.mock.calls[0][0]), "http://localhost");
    expect(url.searchParams.get("category_id")).toBe("cat-1");
    expect(url.searchParams.get("risk_level")).toBe("high,critical");
  });

  it("returns an empty page without calling the backend when calculator filters exclude every value", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const result = await calculatorService.listTable(
      {
        page: 1,
        perPage: 20,
        search: "",
        filters: [
          { id: "1", field: "type", condition: "equals", value: "calculator" },
        ],
      },
      "checklist",
    );

    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.totalItems).toBe(0);
  });

  it("maps legacy drug form fields to the typed drug DTO", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json(
        {
          id: "drug-1",
          name: "Example",
          status: "active",
          review_status: "approved",
        },
        201,
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await drugService.create({
      name: "Example",
      categories: ["category-1"],
      tags: ["tag-1"],
      drug_class: "class-1",
      therapeutic_category: "therapy-1",
      references: "Clinical reference",
    });

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(String(init.body))).toMatchObject({
      categories: ["category-1"],
      tags: ["tag-1"],
      drug_class_id: "class-1",
      therapeutic_category_id: "therapy-1",
      reference_text: "Clinical reference",
    });
  });

  it("loads active drug references without PocketBase filter expressions", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      json({
        items: [],
        page: 1,
        per_page: 100,
        total_items: 0,
        total_pages: 0,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await drugReferenceService.allCategories();

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain("/api/v2/drug-categories");
    expect(url).toContain("status=active");
    expect(url).not.toContain("filter=");
  });
});
