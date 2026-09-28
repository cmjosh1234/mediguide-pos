import { beforeEach, describe, expect, it, vi } from "vitest";

const send = vi.fn();
vi.mock("@/lib/backend-client", () => ({
  getBackendClient: () => ({ send }),
}));

import {
  contentDiseaseService,
  contentHubService,
  diseaseService,
  type ContentPillar,
} from "./content-hubs.service";

describe("disease and content-hub administration services", () => {
  beforeEach(() => send.mockReset());

  it("uses typed disease taxonomy and assignment endpoints", async () => {
    send.mockResolvedValueOnce({ id: "disease-1" }).mockResolvedValueOnce([]);

    await diseaseService.create({
      name: "Cholera",
      slug: "cholera",
      status: "active",
      sort_order: 10,
      aliases: [{ alias: "Vibrio cholerae infection" }],
      codes: [{ code_system: "ICD-10", code: "A00" }],
    });
    await contentDiseaseService.replace(
      "guideline",
      "guideline-1",
      ["disease-1", "disease-2"],
      "disease-1",
    );

    expect(send).toHaveBeenNthCalledWith(1, "/api/v2/diseases", {
      method: "POST",
      body: expect.stringContaining('"slug":"cholera"'),
    });
    expect(send).toHaveBeenNthCalledWith(
      2,
      "/api/v2/content-disease-assignments/replace",
      {
        method: "PUT",
        body: JSON.stringify({
          content_type: "guideline",
          content_id: "guideline-1",
          disease_ids: ["disease-1", "disease-2"],
          primary_disease_id: "disease-1",
        }),
      },
    );
  });

  it("applies templates and sends deterministic optimistic-lock reordering", async () => {
    const pillars = [
      { id: "second", lock_version: 3 },
      { id: "first", lock_version: 8 },
    ] as ContentPillar[];
    send.mockResolvedValue([]);

    await contentHubService.applyTemplate("hub-1", "template-1", 4);
    await contentHubService.reorderPillars("hub-1", pillars);

    expect(send).toHaveBeenNthCalledWith(
      1,
      "/api/v2/content-hubs/hub-1/apply-template",
      {
        method: "POST",
        body: JSON.stringify({ template_id: "template-1", lock_version: 4 }),
      },
    );
    expect(send).toHaveBeenNthCalledWith(
      2,
      "/api/v2/content-hubs/hub-1/pillars/reorder",
      {
        method: "PUT",
        body: JSON.stringify([
          { id: "second", sort_order: 10, lock_version: 3 },
          { id: "first", sort_order: 20, lock_version: 8 },
        ]),
      },
    );
  });

  it("keeps resource type and outbreak filtering explicit", async () => {
    send.mockResolvedValue({ items: [] });
    await contentHubService.searchResources(
      "situation_report",
      "triage",
      "outbreak-1",
    );
    expect(send).toHaveBeenCalledWith("/api/v2/content-hub-resources", {
      query: {
        page: 1,
        per_page: 50,
        content_type: "situation_report",
        search: "triage",
        outbreak_id: "outbreak-1",
      },
    });
  });
});
