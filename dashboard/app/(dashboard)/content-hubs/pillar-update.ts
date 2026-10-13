import type { ContentPillar } from "@/services/content-hubs.service";

type PillarChanges = Partial<Pick<ContentPillar, "name" | "icon" | "parent_id" | "status">> & { clear_parent?: boolean };

/**
 * The body for updating a pillar. The API replaces every field with what it
 * is sent, so the fields not being changed are sent as currently stored.
 */
export function pillarUpdate(pillar: ContentPillar, changes: PillarChanges) {
  return {
    name: pillar.name,
    slug: pillar.slug,
    description: pillar.description || "",
    icon: pillar.icon || "",
    color: pillar.color || "",
    sort_order: pillar.sort_order,
    status: pillar.status,
    parent_id: pillar.parent_id || undefined,
    lock_version: pillar.lock_version,
    ...changes,
  };
}
