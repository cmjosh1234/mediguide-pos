import { expect, test } from "vitest";

import type { HubAudit } from "@/services/content-hubs.service";

import { hubActionLabel, hubAuditDetails } from "./hub-audit-history";

const entry = (action: string, entity_type: string, metadata: Record<string, unknown>) =>
  ({ id: "a1", actor_id: "u1", action, entity_type, entity_id: "e1", metadata, created_at: "2026-10-12T10:00:00Z" }) as HubAudit;

test("a pillar deletion names the pillar, what went with it and why", () => {
  const deleted = entry("content_pillar.delete", "content_pillar", { name: "Treatment", reason: "Merged", removed_sub_pillars: 1, removed_assignments: 3 });
  expect(hubActionLabel(deleted)).toBe("Pillar deleted");
  expect(hubAuditDetails(deleted, new Map())).toEqual([
    ["Pillar", "Treatment"],
    ["Also removed", "1 sub-pillar and 3 resource assignments"],
    ["Reason", "Merged"],
  ]);
});

test("archiving, assignments and templates read as plain actions", () => {
  expect(hubActionLabel(entry("content_pillar.update", "content_pillar", { name: "Overview", status: "archived" }))).toBe("Pillar archived");
  const activated = entry("content_pillar_item.update", "content_pillar_item", { label_override: "UCG", status: "active" });
  expect(hubActionLabel(activated)).toBe("Assignment edited");
  expect(hubAuditDetails(activated, new Map())).toEqual([["Resource", "UCG"], ["Status", "Active"]]);
  const applied = entry("content_hub.apply_template", "content_hub", { template_id: "t1" });
  expect(hubActionLabel(applied)).toBe("Template applied");
  expect(hubAuditDetails(applied, new Map([["t1", "Disease care"]]))).toEqual([["Template", "Disease care"]]);
});

test("status moves read as archive and unarchive; saving an archived pillar is an edit", () => {
  const restored = entry("content_pillar.update", "content_pillar", { name: "IPC & PPE", status: "active", from_status: "archived", to_status: "active" });
  expect(hubActionLabel(restored)).toBe("Pillar unarchived");
  expect(hubAuditDetails(restored, new Map())).toEqual([["Pillar", "IPC & PPE"], ["Status", "Archived → Active"]]);
  expect(hubActionLabel(entry("content_pillar.update", "content_pillar", { status: "archived", from_status: "active", to_status: "archived" }))).toBe("Pillar archived");
  expect(hubActionLabel(entry("content_pillar.update", "content_pillar", { status: "archived", from_status: "archived", to_status: "archived" }))).toBe("Pillar edited");
});

test("a template replacement names the template, what it removed and why", () => {
  const replaced = entry("content_hub.apply_template", "content_hub", { template_id: "t2", template_name: "Surveillance knowledge", replace: true, reason: "Surveillance layout", removed_pillars: 9, removed_assignments: 1 });
  expect(hubActionLabel(replaced)).toBe("Template replaced");
  expect(hubAuditDetails(replaced, new Map())).toEqual([
    ["Template", "Surveillance knowledge"],
    ["Removed", "9 pillars and 1 resource assignment"],
    ["Reason", "Surveillance layout"],
  ]);
});

test("hub status moves name a published hub as published", () => {
  const unarchived = entry("content_hub.unarchive", "content_hub", { from_status: "archived", to_status: "draft" });
  expect(hubActionLabel(unarchived)).toBe("Hub unarchived");
  expect(hubAuditDetails(unarchived, new Map())).toEqual([["Status", "Archived → Draft"]]);
  const archived = entry("content_hub.archive", "content_hub", { from_status: "active", to_status: "archived" });
  expect(hubAuditDetails(archived, new Map())).toEqual([["Status", "Published → Archived"]]);
});
