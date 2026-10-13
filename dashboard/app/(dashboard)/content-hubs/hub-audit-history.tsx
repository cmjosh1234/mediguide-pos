"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { contentHubService, type HubAudit, type HubTemplate, type Page } from "@/services/content-hubs.service";

import { AuditEntry } from "../outbreaks/audit-history";
import { DEFAULT_PAGE_SIZE, PaginationControls } from "../outbreaks/list-pagination";

const ACTIONS: Record<string, string> = {
  "content_hub.create": "Hub created",
  "content_hub.update": "Hub details edited",
  "content_hub.publish": "Hub published",
  "content_hub.archive": "Hub archived",
  "content_hub.unarchive": "Hub unarchived",
  "content_hub.delete": "Hub deleted",
  "content_hub.apply_template": "Template applied",
  "content_pillar.create": "Pillar added",
  "content_pillar.update": "Pillar edited",
  "content_pillar.delete": "Pillar deleted",
  "content_pillar.reorder": "Pillars reordered",
  "content_pillar_item.create": "Resource assigned",
  "content_pillar_item.update": "Assignment edited",
  "content_pillar_item.delete": "Assignment removed",
  "content_pillar_item.reorder": "Assignments reordered",
};

/**
 * The paged audit trail of a content hub, its pillars and their assignments,
 * laid out like the outbreak audit history. Reloads from the first page
 * whenever refreshKey changes.
 */
export function HubAuditHistory({ hubId, templates, refreshKey }: { hubId: string; templates: HubTemplate[]; refreshKey?: unknown }) {
  const [page, setPage] = React.useState(1);
  const [history, setHistory] = React.useState<Page<HubAudit> | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState("");

  React.useEffect(() => setPage(1), [refreshKey]);
  React.useEffect(() => {
    let current = true;
    setLoading(true);
    setError("");
    contentHubService
      .audit(hubId, page, DEFAULT_PAGE_SIZE)
      .then((result) => { if (current) setHistory(result); })
      .catch((value) => { if (current) setError(value instanceof Error ? value.message : "Unable to load the audit history"); })
      .finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [hubId, page, refreshKey]);

  const templateNames = React.useMemo(() => new Map(templates.map((value) => [value.id, value.name])), [templates]);
  const items = history?.items || [];
  return (
    <Card>
      <CardHeader>
        <CardTitle>Audit history</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        {loading && !history ? (
          <p className="py-6 text-center text-sm text-muted-foreground"><Loader2 className="mr-2 inline h-4 w-4 animate-spin" />Loading audit history…</p>
        ) : null}
        {history && !items.length ? <p className="text-sm text-muted-foreground">No hub changes have been recorded.</p> : null}
        {items.length ? (
          <ol className={`space-y-2 ${loading ? "opacity-60" : ""}`} aria-label="Audit history" aria-busy={loading}>
            {items.map((entry) => (
              <AuditEntry key={entry.id} entry={entry} label={hubActionLabel(entry)} details={hubAuditDetails(entry, templateNames)} />
            ))}
          </ol>
        ) : null}
        <PaginationControls
          label="Audit history"
          pagination={{
            visible: items,
            offset: (page - 1) * DEFAULT_PAGE_SIZE,
            page,
            pageCount: Math.max(1, history?.total_pages || 1),
            pageSize: DEFAULT_PAGE_SIZE,
            total: history?.total_items || 0,
            setPage,
            showIndex: (index) => setPage(Math.floor(index / DEFAULT_PAGE_SIZE) + 1),
          }}
        />
      </CardContent>
    </Card>
  );
}

export function hubActionLabel(entry: HubAudit) {
  if (entry.action === "content_pillar.update") {
    const { from_status: from, to_status: to, status } = entry.metadata || {};
    if (to === undefined) {
      // Recorded before updates kept the previous status.
      if (status === "archived") return "Pillar archived";
    } else if (from !== to && to === "archived") {
      return "Pillar archived";
    } else if (from === "archived" && to !== "archived") {
      return "Pillar unarchived";
    }
  }
  if (entry.action === "content_hub.apply_template" && entry.metadata?.replace === true) return "Template replaced";
  return ACTIONS[entry.action] || sentence(entry.action.slice(entry.action.indexOf(".") + 1));
}

/** What the entry was about and why, as label and value pairs. */
export function hubAuditDetails(entry: HubAudit, templateNames: Map<string, string>): Array<[string, string]> {
  const meta = entry.metadata || {};
  const text = (value: unknown) => (typeof value === "string" ? value.trim() : "");
  const details: Array<[string, string]> = [];
  if (entry.entity_type === "content_pillar" && text(meta.name)) details.push(["Pillar", text(meta.name)]);
  if (text(meta.from_status) && text(meta.to_status) && meta.from_status !== meta.to_status) {
    const status = (value: string) => (entry.entity_type === "content_hub" && value === "active" ? "Published" : sentence(value));
    details.push(["Status", `${status(text(meta.from_status))} → ${status(text(meta.to_status))}`]);
  }
  if (entry.entity_type === "content_pillar_item") {
    if (text(meta.label_override)) details.push(["Resource", text(meta.label_override)]);
    if (entry.action.endsWith(".update") && text(meta.status)) details.push(["Status", sentence(text(meta.status))]);
  }
  if (entry.action === "content_hub.apply_template" && (text(meta.template_name) || text(meta.template_id))) {
    details.push(["Template", text(meta.template_name) || templateNames.get(text(meta.template_id)) || text(meta.template_id)]);
    const removed = [count(meta.removed_pillars, "pillar"), count(meta.removed_assignments, "resource assignment")].filter(Boolean);
    if (removed.length) details.push(["Removed", removed.join(" and ")]);
  }
  if (entry.action === "content_pillar.delete") {
    const removed = [count(meta.removed_sub_pillars, "sub-pillar"), count(meta.removed_assignments, "resource assignment")].filter(Boolean);
    if (removed.length) details.push(["Also removed", removed.join(" and ")]);
  }
  if (text(meta.reason)) details.push(["Reason", text(meta.reason)]);
  return details;
}

function count(value: unknown, noun: string) {
  return typeof value === "number" && value > 0 ? `${value} ${noun}${value === 1 ? "" : "s"}` : "";
}

function sentence(value: string) {
  const text = value.replace(/_/g, " ");
  return text.charAt(0).toUpperCase() + text.slice(1);
}
