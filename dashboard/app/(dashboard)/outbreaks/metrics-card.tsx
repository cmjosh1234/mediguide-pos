"use client";

// The metrics card and "Add metric" dialog shared by the outbreak and situation
// report editors. Each editor decides how a change is saved (onPersist).

import * as React from "react";
import { Loader2, Plus, Save } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { showToast } from "@/lib/toast";
import { DEFAULT_PAGE_SIZE, PaginationControls, usePagination } from "./list-pagination";
import { Field } from "./resource-fields";

export type MetricDraft = {
  key: string;
  label: string;
  value: string;
  numeric_value?: number;
  unit: string;
  as_of: string;
  source_reference: string;
  sort_order: number;
  _localId: string;
};

type Problem = { id: string; label: string; message: string };

function newMetricId() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);
}

const emptyMetric = (): MetricDraft => ({
  key: "",
  label: "",
  value: "",
  unit: "",
  as_of: new Date().toISOString(),
  source_reference: "",
  sort_order: 1,
  _localId: newMetricId(),
});

// Metrics loaded from or just saved to the backend get a fresh _localId to key
// their rows by.
export function withLocalIds(metrics: Omit<MetricDraft, "_localId">[]): MetricDraft[] {
  return metrics.map((metric) => ({ ...metric, _localId: newMetricId() }));
}

export function stripLocalIds(metrics: MetricDraft[]) {
  return metrics.map(({ key, label, value, numeric_value, unit, as_of, source_reference, sort_order }) => ({
    key,
    label,
    value,
    numeric_value,
    unit,
    as_of,
    source_reference,
    sort_order,
  }));
}

// Mirrors the backend rules that reject a metric (outbreak_validation.go).
const METRIC_KEY_PATTERN = /^[a-z][a-z0-9_]{1,63}$/;

// Derives a metric key from its label ("Confirmed cases (demo)" becomes
// confirmed_cases_demo), adding a numeric suffix if another metric uses it.
function metricKeyFromLabel(label: string, existing: MetricDraft[]) {
  const base = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^[^a-z]+|_+$/g, "")
    .slice(0, 60)
    .replace(/_+$/, "");
  if (!base) return "";
  const taken = new Set(existing.map((metric) => metric.key));
  let key = base;
  for (let suffix = 2; taken.has(key); suffix++) key = `${base}_${suffix}`;
  return key;
}

function findMetricProblems(metric: MetricDraft, existing: MetricDraft[]): Problem[] {
  const problems: Problem[] = [];
  const key = metric.key.trim().toLowerCase();
  if (!key) {
    problems.push({ id: "metric-key", label: "Key", message: "Key is required." });
  } else if (!METRIC_KEY_PATTERN.test(key)) {
    problems.push({
      id: "metric-key",
      label: "Key",
      message: "Key must be lowercase letters, numbers or underscores, starting with a letter.",
    });
  } else if (existing.some((other) => other.key === key)) {
    problems.push({ id: "metric-key", label: "Key", message: "Another metric already uses this key." });
  }
  if (!metric.label.trim()) {
    problems.push({ id: "metric-label", label: "Label", message: "Label is required." });
  }
  if (!metric.value.trim()) {
    problems.push({ id: "metric-value", label: "Value", message: "Value is required." });
  }
  if (!metric.source_reference.trim()) {
    problems.push({ id: "metric-source_reference", label: "Source", message: "Source is required." });
  }
  if (!metric.as_of) {
    problems.push({ id: "metric-as_of", label: "As of", message: "The 'as of' date is required." });
  }
  return problems;
}

export type PersistMetrics = (
  next: MetricDraft[],
  success: { title: string; message: string },
  failureTitle: string,
) => Promise<boolean>;

export function MetricsCard({
  metrics,
  saving,
  onPersist,
  description,
  dialogDescription,
  saveLabel,
  confirmRemoval,
  locked = false,
}: {
  metrics: MetricDraft[];
  saving: boolean;
  /** Saves the whole set with one change; resolves false if it failed. */
  onPersist: PersistMetrics;
  description: React.ReactNode;
  dialogDescription: string;
  saveLabel: string;
  /** Ask before removing, when a removal is saved straight away. */
  confirmRemoval: boolean;
  /** Metrics can be read but not changed (a published report). */
  locked?: boolean;
}) {
  const entries = React.useMemo(() => metrics.map((metric, index) => ({ metric, index })), [metrics]);
  const pages = usePagination(entries, DEFAULT_PAGE_SIZE);
  // The metric being added in the dialog; null while it's closed.
  const [draft, setDraft] = React.useState<MetricDraft | null>(null);
  const [attempted, setAttempted] = React.useState(false);
  // The key follows the label until it's edited by hand.
  const [keyEdited, setKeyEdited] = React.useState(false);
  const problems = React.useMemo(() => (draft ? findMetricProblems(draft, metrics) : []), [draft, metrics]);
  const errorFor = (id: string) =>
    attempted ? problems.find((problem) => problem.id === id)?.message : undefined;

  function open() {
    setAttempted(false);
    setKeyEdited(false);
    setDraft(emptyMetric());
  }

  function update(key: keyof MetricDraft, value: string) {
    setDraft((current) => {
      if (!current) return current;
      const next = { ...current, [key]: value };
      if (key === "label" && !keyEdited) next.key = metricKeyFromLabel(value, metrics);
      return next;
    });
  }

  async function add() {
    if (!draft) return;
    if (problems.length > 0) {
      setAttempted(true);
      showToast.error(
        "Missing required information",
        `Fill in: ${problems.map((problem) => problem.label).join(", ")}`,
        { richColors: true },
      );
      document.getElementById(problems[0].id)?.focus();
      return;
    }
    const sortOrder = Math.max(0, ...metrics.map((metric) => metric.sort_order)) + 1;
    const added = await onPersist(
      [...metrics, { ...draft, key: draft.key.trim().toLowerCase(), sort_order: sortOrder }],
      { title: "Metric saved", message: `${draft.label.trim()} was added.` },
      "Unable to save metric",
    );
    if (added) setDraft(null);
  }

  async function remove(index: number) {
    if (confirmRemoval && !window.confirm("Remove this metric? It will be deleted immediately.")) return;
    await onPersist(
      metrics.filter((_, position) => position !== index),
      { title: "Metric removed", message: "The metric was deleted." },
      "Unable to remove metric",
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Metrics</CardTitle>
            <Button variant="outline" size="sm" disabled={saving || locked} onClick={open}>
              <Plus className="mr-2 h-4 w-4" />
              Metric
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">{description}</p>
        </CardHeader>
        <CardContent className="space-y-3">
          {metrics.length === 0 ? (
            <p className="text-sm text-muted-foreground">No metrics added.</p>
          ) : (
            pages.visible.map(({ metric, index }) => (
              <div
                key={metric._localId}
                className="flex items-start justify-between gap-4 rounded-md border p-3"
              >
                <div className="min-w-0 space-y-1">
                  <p className="font-medium">
                    {metric.label}{" "}
                    <span className="font-mono text-xs font-normal text-muted-foreground">{metric.key}</span>
                  </p>
                  <p className="text-sm">
                    {metric.value}
                    {metric.unit ? ` ${metric.unit}` : ""}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {metric.source_reference}
                    {metric.as_of ? ` · as of ${new Date(metric.as_of).toLocaleString()}` : ""}
                  </p>
                </div>
                {locked ? null : (
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={saving}
                    onClick={() => void remove(index)}
                  >
                    Remove
                  </Button>
                )}
              </div>
            ))
          )}
          <PaginationControls pagination={pages} label="Metrics" />
        </CardContent>
      </Card>
      <Dialog
        open={Boolean(draft)}
        onOpenChange={(isOpen) => {
          if (!isOpen && !saving) setDraft(null);
        }}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Add metric</DialogTitle>
            <DialogDescription>{dialogDescription}</DialogDescription>
          </DialogHeader>
          {draft ? (
            <div className="grid gap-3 md:grid-cols-2">
              <Field id="metric-label" label="Label" required error={errorFor("metric-label")}>
                <Input
                  id="metric-label"
                  aria-required
                  aria-invalid={Boolean(errorFor("metric-label"))}
                  placeholder="Confirmed cases"
                  value={draft.label}
                  onChange={(event) => update("label", event.target.value)}
                  autoFocus
                />
              </Field>
              <Field id="metric-key" label="Key" required error={errorFor("metric-key")}>
                <Input
                  id="metric-key"
                  className="font-mono"
                  aria-required
                  aria-invalid={Boolean(errorFor("metric-key"))}
                  placeholder="Generated from the label"
                  value={draft.key}
                  onChange={(event) => {
                    // Clearing the key hands it back to the label.
                    setKeyEdited(event.target.value !== "");
                    update("key", event.target.value);
                  }}
                />
              </Field>
              <Field id="metric-value" label="Value" required error={errorFor("metric-value")}>
                <Input
                  id="metric-value"
                  aria-required
                  aria-invalid={Boolean(errorFor("metric-value"))}
                  placeholder="20"
                  value={draft.value}
                  onChange={(event) => update("value", event.target.value)}
                />
              </Field>
              <Field id="metric-unit" label="Unit">
                <Input
                  id="metric-unit"
                  placeholder="cases"
                  value={draft.unit}
                  onChange={(event) => update("unit", event.target.value)}
                />
              </Field>
              <Field id="metric-source_reference" label="Source" required error={errorFor("metric-source_reference")}>
                <Input
                  id="metric-source_reference"
                  aria-required
                  aria-invalid={Boolean(errorFor("metric-source_reference"))}
                  placeholder="WHO situation report 11"
                  value={draft.source_reference}
                  onChange={(event) => update("source_reference", event.target.value)}
                />
              </Field>
              <Field id="metric-as_of" label="As of" required error={errorFor("metric-as_of")}>
                <Input
                  id="metric-as_of"
                  type="datetime-local"
                  aria-required
                  aria-invalid={Boolean(errorFor("metric-as_of"))}
                  value={toLocal(draft.as_of)}
                  onChange={(event) => update("as_of", iso(event.target.value) || "")}
                />
              </Field>
            </div>
          ) : null}
          <DialogFooter>
            <Button variant="outline" disabled={saving} onClick={() => setDraft(null)}>
              Cancel
            </Button>
            <Button disabled={saving} onClick={() => void add()}>
              {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Save className="mr-2 h-4 w-4" />}
              {saveLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function iso(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}

function toLocal(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}
