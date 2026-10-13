import { expect, test } from "vitest";

import type { ContentPillar } from "@/services/content-hubs.service";

import { pillarUpdate } from "./pillar-update";

const overview = {
  id: "p1", hub_id: "h1", parent_id: "p0", name: "Overview", slug: "overview",
  description: "Screening and diagnosis overview.", icon: "book-open", color: "clinical",
  sort_order: 10, status: "active", lock_version: 4, items: [],
} as ContentPillar;

const stored = {
  name: "Overview", slug: "overview", description: "Screening and diagnosis overview.", icon: "book-open",
  color: "clinical", sort_order: 10, status: "active", parent_id: "p0", lock_version: 4,
};

test("archiving sends the stored fields, including the required name, with only the status changed", () => {
  expect(pillarUpdate(overview, { status: "archived" })).toEqual({ ...stored, status: "archived" });
});

test("saving a row keeps the description, colour, order, slug and status it does not edit", () => {
  expect(pillarUpdate(overview, { name: "Overview and screening", icon: "list", parent_id: undefined, clear_parent: true })).toEqual({
    ...stored, name: "Overview and screening", icon: "list", parent_id: undefined, clear_parent: true,
  });
});
