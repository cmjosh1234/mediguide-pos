"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  AlertCircle,
  CheckCircle2,
  ExternalLink,
  FolderKanban,
  Loader2,
  Plus,
  Save,
  ShieldCheck,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
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
import { Label } from "@/components/ui/label";
import { MultiSelect } from "@/components/ui/multi-select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { apiUrl, getCurrentUser } from "@/lib/backend-client";
import { showToast } from "@/lib/toast";
import { withDashboardBasePath } from "@/lib/dashboard-path";
import { healthFacilitiesService } from "@/services/health-facilities.service";
import {
  outbreaksService,
  type OutbreakAuditRecord,
  type OutbreakRecord,
  type OutbreakResourceRecord,
  type OutbreakUpdateRecord,
  type PublishedGuidelineRecord,
} from "@/services/outbreaks.service";
import {
  situationReportsService,
  type SituationReportRecord,
} from "@/services/situation-reports.service";
import { contentHubService, diseaseService } from "@/services/content-hubs.service";
import { documentKindService, type DocumentKind } from "@/services/document-kinds.service";
import type { DistrictsResponse, RegionsResponse } from "@/types/backend-types";
import { RequiredMark } from "./required-field";
import { CampaignDraftBuilder } from "./campaign-draft-builder";
import { ChildContentWorkflow } from "./child-content-workflow";
import {
  emptyResourceDraft,
  findResourceProblems,
  ResourceFields,
  resourcePayload,
  type ResourceDraft,
} from "./resource-fields";
import { DEFAULT_PAGE_SIZE, PaginationControls, usePagination } from "./list-pagination";

type MetricDraft = {
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
function withLocalIds(metrics: MetricDraft[]): MetricDraft[] {
  return metrics.map((metric) => ({ ...metric, _localId: newMetricId() }));
}

function stripLocalIds(metrics: MetricDraft[]) {
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

type Problem = { id: string; label: string; message: string };

// Mirrors the backend rules that reject a draft save (outbreak_validation.go).
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

function findProblems(form: { title: string; disease_id: string }): Problem[] {
  const problems: Problem[] = [];
  const title = form.title.trim();
  if (!title) {
    problems.push({ id: "outbreak-title", label: "Title", message: "Title is required." });
  } else if (title.length > 240) {
    problems.push({ id: "outbreak-title", label: "Title", message: "Title must be 240 characters or fewer." });
  }
  if (!form.disease_id) {
    problems.push({ id: "outbreak-disease", label: "Disease", message: "Disease is required." });
  }
  return problems;
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

function findUpdateProblems(draft: { title: string }): Problem[] {
  return draft.title.trim()
    ? []
    : [{ id: "update-title", label: "Title", message: "Title is required." }];
}

const WORKFLOW_STEPS = ["Draft", "In review", "Approved", "Published"];

// -1 means withdrawn (outside the forward path).
function workflowStage(status?: string, approvedAt?: string) {
  if (!status || status === "draft") return 0;
  if (status === "pending_review") return approvedAt ? 2 : 1;
  if (status === "withdrawn") return -1;
  return 3;
}

function statusLabel(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function workflowHint(status?: string, approvedAt?: string) {
  switch (workflowStage(status, approvedAt)) {
    case 0:
      return "Fill in the details below and save the draft, then submit it for review.";
    case 1:
      return "Waiting for a reviewer to approve this outbreak.";
    case 2:
      return "Approved. Publish it to make it public.";
    case 3:
      return "Live on the public API. To change it, create a correction.";
    default:
      return "Withdrawn. This outbreak is no longer public.";
  }
}

function WorkflowSteps({ stage }: { stage: number }) {
  if (stage < 0) return <Badge variant="destructive">Withdrawn</Badge>;
  return (
    <ol className="flex items-center gap-2 text-sm">
      {WORKFLOW_STEPS.map((step, index) => {
        const done = index < stage || stage === WORKFLOW_STEPS.length - 1;
        const current = index === stage && !done;
        return (
          <li key={step} className="flex items-center gap-2">
            <span
              className={cn(
                "flex h-6 w-6 items-center justify-center rounded-full border text-xs font-medium",
                done && "border-primary bg-primary text-primary-foreground",
                current && "border-primary text-primary ring-2 ring-primary/30",
                !done && !current && "text-muted-foreground",
              )}
              aria-hidden="true"
            >
              {done ? <CheckCircle2 className="h-3.5 w-3.5" /> : index + 1}
            </span>
            <span
              className={cn(
                current ? "font-medium" : "text-muted-foreground",
              )}
              aria-current={current ? "step" : undefined}
            >
              {step}
            </span>
            {index < WORKFLOW_STEPS.length - 1 ? (
              <span className="h-px w-6 bg-border" aria-hidden="true" />
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}

export function OutbreakEditor({ id }: { id?: string }) {
  const router = useRouter();
  const [item, setItem] = React.useState<OutbreakRecord | null>(null);
  const [updates, setUpdates] = React.useState<OutbreakUpdateRecord[]>([]);
  const [resources, setResources] = React.useState<OutbreakResourceRecord[]>(
    [],
  );
  const [audit, setAudit] = React.useState<OutbreakAuditRecord[]>([]);
  const [reports, setReports] = React.useState<SituationReportRecord[]>([]);
  const [guidelines, setGuidelines] = React.useState<
    PublishedGuidelineRecord[]
  >([]);
  const [regions, setRegions] = React.useState<RegionsResponse[]>([]);
  const [districts, setDistricts] = React.useState<DistrictsResponse[]>([]);
  const [diseases, setDiseases] = React.useState<
    Array<{ id: string; name: string; parent_name?: string; color?: string }>
  >([]);
  const [reviewComment, setReviewComment] = React.useState("");
  const [metrics, setMetrics] = React.useState<MetricDraft[]>([]);
  const metricEntries = React.useMemo(
    () => metrics.map((metric, index) => ({ metric, index })),
    [metrics],
  );
  const metricPages = usePagination(metricEntries, DEFAULT_PAGE_SIZE);
  // The metric being added in the "+ Metric" dialog; null while it's closed.
  const [metricDraft, setMetricDraft] = React.useState<MetricDraft | null>(null);
  const [metricAttempted, setMetricAttempted] = React.useState(false);
  // The key follows the label until it's edited by hand.
  const [metricKeyEdited, setMetricKeyEdited] = React.useState(false);
  const [updateDraft, setUpdateDraft] = React.useState({
    title: "",
    summary: "",
  });
  const [resourceDraft, setResourceDraft] = React.useState<ResourceDraft>(emptyResourceDraft);
  const [documentKinds, setDocumentKinds] = React.useState<DocumentKind[]>([]);
  const [form, setForm] = React.useState({
    title: "",
    disease_id: "",
    geographic_area: "",
    region_id: "",
    district_id: "",
    summary: "",
    visual_tone: "warning",
    source_organization: "",
    source_url: "",
    source_reference: "",
    start_date: "",
    last_update: "",
    effective_at: "",
    data_as_of: "",
    last_verified_at: "",
    change_summary: "",
  });
  const [loading, setLoading] = React.useState(Boolean(id));
  const [saving, setSaving] = React.useState(false);
  const [savingMetrics, setSavingMetrics] = React.useState(false);
  const [error, setError] = React.useState("");

  const hydrate = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError("");
    try {
      const [
        record,
        updatePage,
        resourcePage,
        auditPage,
        reportPage,
        guidelinePage,
      ] = await Promise.all([
        outbreaksService.get(id),
        outbreaksService.listUpdates(id),
        outbreaksService.listResources(id),
        outbreaksService.audit(id),
        situationReportsService.list({
          outbreak_id: id,
          page: 1,
          per_page: 100,
        }),
        outbreaksService.listPublishedGuidelines(),
      ]);
      setItem(record);
      setUpdates(updatePage.items || []);
      setResources(resourcePage.items || []);
      setAudit(auditPage.items || []);
      setReports(reportPage.items || []);
      setGuidelines(guidelinePage.items || []);
      const loadedMetrics = withLocalIds((record.metrics || []) as MetricDraft[]);
      setMetrics(loadedMetrics);
      setForm({
        title: record.title || "",
        disease_id: record.disease_id || "",
        geographic_area: record.geographic_area || "",
        region_id: record.region_id || "",
        district_id: record.district_id || "",
        summary: record.summary || "",
        visual_tone: record.visual_tone || "warning",
        source_organization: record.source_organization || "",
        source_url: record.source_url || "",
        source_reference: record.source_reference || "",
        start_date: toLocal(record.start_date),
        last_update: toLocal(record.last_update),
        effective_at: toLocal(record.effective_at),
        data_as_of: toLocal(record.data_as_of),
        last_verified_at: toLocal(record.last_verified_at),
        change_summary: "",
      });
    } catch (value) {
      setError(
        value instanceof Error
          ? value.message
          : "Unable to load outbreak workspace",
      );
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    void hydrate();
  }, [hydrate]);
  // Regions are needed on the create form too, where hydrate() has no id to load.
  React.useEffect(() => {
    void healthFacilitiesService
      .regions()
      .then(setRegions)
      .catch(() => setRegions([]));
  }, []);
  React.useEffect(() => {
    void diseaseService
      .list("", "active")
      .then((page) => setDiseases(page.items || []))
      .catch(() => setDiseases([]));
  }, []);
  React.useEffect(() => {
    if (!form.region_id) {
      setDistricts([]);
      return;
    }
    void healthFacilitiesService
      .districts(form.region_id)
      .then(setDistricts)
      .catch(() => setDistricts([]));
  }, [form.region_id]);

  function field(name: keyof typeof form, value: string) {
    setForm((current) => ({ ...current, [name]: value }));
  }
  const immutable =
    item?.status === "published" ||
    item?.status === "active" ||
    item?.status === "monitoring" ||
    item?.status === "contained" ||
    item?.status === "closed" ||
    item?.status === "withdrawn";

  // Problems are always computed, but only shown once a save has been attempted,
  // so the form isn't covered in errors before the user has typed anything.
  const [attempted, setAttempted] = React.useState(false);
  const problems = React.useMemo(
    () => findProblems(form),
    [form],
  );
  const visibleProblems = attempted ? problems : [];
  const errorFor = (id: string) =>
    visibleProblems.find((problem) => problem.id === id)?.message;
  const metricProblems = React.useMemo(
    () => (metricDraft ? findMetricProblems(metricDraft, metrics) : []),
    [metricDraft, metrics],
  );
  const metricError = (id: string) =>
    metricAttempted
      ? metricProblems.find((problem) => problem.id === id)?.message
      : undefined;

  function focusField(id: string) {
    // Wait for the field to render before focusing it.
    window.setTimeout(() => {
      const element = document.getElementById(id);
      if (!element) return;
      element.scrollIntoView({ block: "center", behavior: "smooth" });
      element.focus({ preventScroll: true });
    }, 0);
  }

  const [updateAttempted, setUpdateAttempted] = React.useState(false);
  const [resourceAttempted, setResourceAttempted] = React.useState(false);
  const [statusTarget, setStatusTarget] = React.useState("monitoring");
  const updateProblems = React.useMemo(
    () => findUpdateProblems(updateDraft),
    [updateDraft],
  );
  const resourceProblems = React.useMemo(
    () => findResourceProblems(resourceDraft),
    [resourceDraft],
  );
  React.useEffect(() => {
    void documentKindService
      .list()
      .then(setDocumentKinds)
      .catch(() => setDocumentKinds([]));
  }, []);
  const updateError = (id: string) =>
    updateAttempted
      ? updateProblems.find((problem) => problem.id === id)?.message
      : undefined;
  const resourceError = (id: string) =>
    resourceAttempted
      ? resourceProblems.find((problem) => problem.id === id)?.message
      : undefined;

  function rejectIncomplete(problems: Problem[]) {
    showToast.error(
      "Missing required information",
      `Fill in: ${problems.map((problem) => problem.label).join(", ")}`,
      { richColors: true },
    );
    focusField(problems[0].id);
  }

  async function save() {
    if (immutable) {
      showToast.error(
        "Published content",
        "Create a correction instead of mutating published content.",
      );
      return;
    }
    if (problems.length > 0) {
      setAttempted(true);
      const missing = problems
        .map((problem) => problem.label)
        .filter((label, i, all) => all.indexOf(label) === i);
      showToast.error(
        "Missing required information",
        `Fill in: ${missing.join(", ")}`,
        { richColors: true },
      );
      focusField(problems[0].id);
      return;
    }
    setSaving(true);
    try {
      const payload = {
        ...form,
        disease_id: form.disease_id || undefined,
        region_id: form.region_id || undefined,
        district_id: form.district_id || undefined,
        change_summary: undefined,
        start_date: iso(form.start_date),
        last_update: iso(form.last_update),
        effective_at: iso(form.effective_at),
        data_as_of: iso(form.data_as_of),
        last_verified_at: iso(form.last_verified_at),
        metrics: stripLocalIds(metrics),
        ...(item ? { lock_version: item.lock_version } : {}),
      };
      const saved = item
        ? await outbreaksService.update(item.id!, payload)
        : await outbreaksService.create(payload);
      showToast.success(
        "Outbreak saved",
        "The draft was saved without publishing it.",
      );
      if (!item) {
        window.location.assign(withDashboardBasePath(`/outbreaks/${saved.id}`));
        return;
      }
      setItem(saved);
      setMetrics(withLocalIds((saved.metrics || []) as MetricDraft[]));
    } catch (value) {
      showToast.error("Unable to save", conflictMessage(value), {
        richColors: true,
      });
    } finally {
      setSaving(false);
    }
  }

  function openMetricDialog() {
    setMetricAttempted(false);
    setMetricKeyEdited(false);
    setMetricDraft(emptyMetric());
  }

  // The backend only replaces the whole metric set, so adding or removing one
  // metric sends the current set with that single change. Once an outbreak
  // exists each change is saved immediately; on the create form the metrics
  // are kept locally and sent with the first save.
  async function persistMetrics(
    next: MetricDraft[],
    success: { title: string; message: string },
    failureTitle: string,
  ) {
    if (!item) {
      setMetrics(next);
      return true;
    }
    setSavingMetrics(true);
    try {
      const saved = await outbreaksService.updateMetrics(item.id!, {
        metrics: stripLocalIds(next),
        lock_version: item.lock_version,
      });
      showToast.success(success.title, success.message);
      setItem(saved);
      setMetrics(withLocalIds((saved.metrics || []) as MetricDraft[]));
      return true;
    } catch (value) {
      showToast.error(failureTitle, conflictMessage(value), {
        richColors: true,
      });
      return false;
    } finally {
      setSavingMetrics(false);
    }
  }

  async function addMetric() {
    if (!metricDraft) return;
    if (metricProblems.length > 0) {
      setMetricAttempted(true);
      rejectIncomplete(metricProblems);
      return;
    }
    const sortOrder = Math.max(0, ...metrics.map((metric) => metric.sort_order)) + 1;
    const added = await persistMetrics(
      [
        ...metrics,
        { ...metricDraft, key: metricDraft.key.trim().toLowerCase(), sort_order: sortOrder },
      ],
      { title: "Metric saved", message: `${metricDraft.label.trim()} was added.` },
      "Unable to save metric",
    );
    if (added) setMetricDraft(null);
  }

  async function removeMetric(index: number) {
    if (item && !window.confirm("Remove this metric? It will be deleted immediately."))
      return;
    await persistMetrics(
      metrics.filter((_, position) => position !== index),
      { title: "Metric removed", message: "The metric was deleted." },
      "Unable to remove metric",
    );
  }

  async function workflow(
    action: "submit" | "approve" | "publish" | "withdraw" | "correct",
  ) {
    if (!item) return;
    const reason =
      action === "withdraw" || action === "correct"
        ? window.prompt(`Reason for ${action}`)?.trim()
        : "";
    if ((action === "withdraw" || action === "correct") && !reason) return;
    if ( action === "approve" && !window.confirm("Approve this outbreak for publication?")) return;
    if (
      action === "publish" &&
      !window.confirm(
        "Publish this verified outbreak to the public API? This cannot be edited in place.",
      )
    )
      return;
    setSaving(true);
    try {
      const next =
        action === "correct"
          ? await outbreaksService.correct(item.id!, {
              lock_version: item.lock_version!,
              reason,
            })
          : await outbreaksService.transition(item.id!, action, {
              lock_version: item.lock_version!,
              reason,
            });
      setItem(next);
      showToast.success("Workflow updated", `Outbreak is now ${next.status}.`);
      if (action === "correct")
        window.location.assign(withDashboardBasePath(`/outbreaks/${next.id}`));
    } catch (value) {
      showToast.error("Workflow failed", conflictMessage(value), {
        richColors: true,
      });
    } finally {
      setSaving(false);
    }
  }

  async function changeStatus(target: string) {
    if (!item) return;
    const reason = window.prompt(
      `Reason for moving this outbreak to ${target}`,
    )?.trim();
    if (!reason) return;
    setSaving(true);
    try {
      const next = await outbreaksService.updateStatus(item.id!, {
        lock_version: item.lock_version!,
        operational_status: target,
        reason,
      });
      setItem(next);
      showToast.success("Status updated", `Outbreak is now ${next.status}.`);
    } catch (value) {
      showToast.error("Status change failed", conflictMessage(value), {
        richColors: true,
      });
    } finally {
      setSaving(false);
    }
  }

  async function configureContentHub() {
    if (!item?.id) return;
    setSaving(true);
    try {
      const workspace = await contentHubService.configureOutbreak(item.id);
      showToast.success(
        "Outbreak hub ready",
        "Published resources were mapped into editable pillars.",
      );
      router.push(`/content-hubs/${workspace.hub.id}`);
    } catch (value) {
      showToast.error("Hub not configured", conflictMessage(value));
    } finally {
      setSaving(false);
    }
  }

  async function addUpdate() {
    if (!item) return;
    if (updateProblems.length > 0) {
      setUpdateAttempted(true);
      rejectIncomplete(updateProblems);
      return;
    }
    try {
      const created = await outbreaksService.createUpdate(
        item.id!,
        updateDraft,
      );
      setUpdates((current) => [...current, created]);
      setUpdateDraft({ title: "", summary: "" });
      setUpdateAttempted(false);
      showToast.success(
        "Update draft created",
        "Submit it independently when it is ready.",
      );
    } catch (value) {
      showToast.error("Update not created", conflictMessage(value));
    }
  }

  async function addResource() {
    if (!item) return;
    if (resourceProblems.length > 0) {
      setResourceAttempted(true);
      rejectIncomplete(resourceProblems);
      return;
    }
    try {
      const created = await outbreaksService.createResource(item.id!, {
        ...resourcePayload(resourceDraft),
        sort_order: resources.length + 1,
      });
      setResources((current) => [...current, created]);
      setResourceDraft(emptyResourceDraft());
      setResourceAttempted(false);
      showToast.success(
        "Resource draft created",
        "Submit it for review when it is ready.",
      );
    } catch (value) {
      showToast.error("Resource not created", conflictMessage(value));
    }
  }

  async function addReviewComment() {
    if (!item?.id || !reviewComment.trim()) return;
    try {
      await outbreaksService.addReviewComment(item.id, reviewComment);
      setReviewComment("");
      const history = await outbreaksService.audit(item.id);
      setAudit(history.items || []);
      showToast.success(
        "Review comment added",
        "The comment is recorded in audit history.",
      );
    } catch (value) {
      showToast.error("Comment not added", conflictMessage(value));
    }
  }

  if (loading)
    return (
      <div className="py-20 text-center">
        <Loader2 className="mr-2 inline h-5 w-5 animate-spin" />
        Loading workspace…
      </div>
    );
  if (error)
    return (
      <div
        className="rounded-md border border-destructive/40 p-5 text-destructive"
        role="alert"
      >
        {error}
        <Button
          className="ml-3"
          variant="outline"
          onClick={() => void hydrate()}
        >
          Retry
        </Button>
      </div>
    );

  // Backend governance rules (TransitionOutbreak): authors cannot approve their
  // own outbreak, only published outbreaks can be withdrawn, and a critical
  // outbreak cannot be published by the person who approved it.
  const currentUserId = String(getCurrentUser()?.id ?? "");
  const isAuthor = Boolean(
    item?.author_id && item.author_id === currentUserId,
  );
  const isPublished = immutable && item?.status !== "withdrawn";
  // Resources can keep being added while an outbreak is live; only closed and
  // withdrawn outbreaks are finished.
  const resourcesLocked = item?.status === "closed" || item?.status === "withdrawn";
  // active/monitoring/contained/published can move to any of the others, or to
  // closed. closed and withdrawn are both terminal (backend: update_status).
  const canChangeStatus = isPublished && item?.status !== "closed";
  const statusOptions = ["published", "active", "monitoring", "contained", "closed"].filter(
    (value) => value !== item?.status,
  );
  const selectedStatusTarget = statusOptions.includes(statusTarget)
    ? statusTarget
    : statusOptions[0];
  const approvedByMe = Boolean(
    item?.approved_by && item.approved_by === currentUserId,
  );
  const publishBlockedByRole = item?.visual_tone === "critical" && approvedByMe;
  // Saved values only: the backend validates the stored record, not the form.
  const missingForPublish = item
    ? [
        ["Geographic coverage", item.geographic_area],
        ["Source organization", item.source_organization],
        ["Source reference", item.source_reference],
        ["Effective at", item.effective_at],
        ["Last verified", item.last_verified_at],
      ]
        .filter(([, value]) => !String(value ?? "").trim())
        .map(([label]) => label)
    : [];
  const inReview = item?.status === "pending_review";
  const workflowNotices = [
    inReview && !item?.approved_at && isAuthor
      ? "You created this outbreak, so a different reviewer has to approve it."
      : null,
    inReview && item?.approved_at && publishBlockedByRole
      ? "Critical outbreaks must be published by someone other than the approver."
      : null,
    inReview && missingForPublish.length > 0
      ? `Before it can be published, fill in and save: ${missingForPublish.join(", ")}.`
      : null,
  ].filter((value): value is string => Boolean(value));

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold">
              {item ? item.title || "Untitled outbreak" : "New outbreak"}
            </h1>
            {item ? <Badge variant="outline">{item.status}</Badge> : null}
          </div>
          <p className="text-sm text-muted-foreground">
            Draft, review, publish, correct and distribute verified outbreak
            content.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" asChild>
            <Link href="/outbreaks">Back to list</Link>
          </Button>
          {item ? (
            <Button
              variant="outline"
              disabled={saving}
              onClick={() => void configureContentHub()}
            >
              <FolderKanban className="mr-2 h-4 w-4" />
              Configure hub
            </Button>
          ) : null}
          <Button disabled={saving || immutable} onClick={() => void save()}>
            <Save className="mr-2 h-4 w-4" />
            Save draft
          </Button>
        </div>
      </div>
      {item ? (
        <Card>
          <CardHeader className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <CardTitle>Publication workflow</CardTitle>
              <Button variant="ghost" size="sm" asChild>
                <a href={apiUrl(`/api/public/outbreaks/${item.id}`)} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Preview public API
                </a>
              </Button>
            </div>
            <WorkflowSteps
              stage={workflowStage(item.status, item.approved_at)}
            />
            <p className="text-sm text-muted-foreground">
              {workflowHint(item.status, item.approved_at)}
            </p>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            <Button
              variant={item.status === "draft" ? "default" : "outline"}
              disabled={saving || item.status !== "draft"}
              onClick={() => void workflow("submit")}
            >
              Submit for review
            </Button>
            <Button
              variant={
                item.status === "pending_review" && !item.approved_at
                  ? "default"
                  : "outline"
              }
              disabled={
                saving ||
                item.status !== "pending_review" ||
                Boolean(item.approved_at) ||
                isAuthor
              }
              onClick={() => void workflow("approve")}
            >
              <CheckCircle2 className="mr-2 h-4 w-4" />
              Approve
            </Button>
            <Button
              variant={
                item.status === "pending_review" && item.approved_at
                  ? "default"
                  : "outline"
              }
              disabled={
                saving ||
                item.status !== "pending_review" ||
                !item.approved_at ||
                publishBlockedByRole ||
                missingForPublish.length > 0
              }
              onClick={() => void workflow("publish")}
            >
              Publish
            </Button>
            <div className="ml-auto flex flex-wrap gap-2">
              <Button
                variant="outline"
                disabled={saving || !immutable || item.status === "withdrawn"}
                onClick={() => void workflow("correct")}
              >
                Create correction
              </Button>
              <Button
                variant="destructive"
                disabled={saving || !isPublished}
                onClick={() => void workflow("withdraw")}
              >
                Withdraw
              </Button>
            </div>
            {canChangeStatus ? (
              <div className="flex w-full flex-wrap items-center gap-2 border-t pt-3">
                <Label htmlFor="status-target" className="text-sm">
                  Change status to
                </Label>
                <select
                  id="status-target"
                  className="h-9 rounded-md border bg-background px-2 text-sm"
                  value={selectedStatusTarget}
                  onChange={(event) => setStatusTarget(event.target.value)}
                >
                  {statusOptions.map((value) => (
                    <option key={value} value={value}>
                      {statusLabel(value)}
                    </option>
                  ))}
                </select>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={saving}
                  onClick={() => void changeStatus(selectedStatusTarget)}
                >
                  Change
                </Button>
                <span className="text-xs text-muted-foreground">
                  A reason is required and is recorded in the audit history.
                </span>
              </div>
            ) : null}
            {workflowNotices.map((notice) => (
              <p
                key={notice}
                className="w-full text-sm text-amber-700 dark:text-amber-400"
              >
                {notice}
              </p>
            ))}
          </CardContent>
        </Card>
      ) : null}
      {immutable ? (
        <div className="flex gap-3 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950">
          <ShieldCheck className="h-5 w-5" />
          <div>
            <strong>Published content is immutable.</strong> Create a correction
            to make changes.
          </div>
        </div>
      ) : null}
      <Card>
        <CardHeader>
          <CardTitle>Core metadata and source</CardTitle>
          <p className="text-sm text-muted-foreground">
            <RequiredMark /> Required to save a draft. Fields marked &ldquo;needed
            to publish&rdquo; can be left empty for now.
          </p>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <Field
            id="outbreak-title"
            label="Title"
            required
            error={errorFor("outbreak-title")}
            value={form.title}
            onChange={(value) => field("title", value)}
            disabled={immutable}
          />
          <div>
            <Label htmlFor="outbreak-disease">
              Disease
              <RequiredMark />
              <span className="sr-only">(required)</span>
            </Label>
            <MultiSelect
              className="mt-2"
              maxSelections={1}
              options={diseases.map((value) => ({
                value: value.id,
                label: `${value.parent_name ? `${value.parent_name} › ` : ""}${value.name}`,
                color: value.color,
              }))}
              value={form.disease_id ? [form.disease_id] : []}
              onValueChange={(value) => field("disease_id", value[0] ?? "")}
              placeholder="Search diseases"
              disabled={immutable}
            />
            {errorFor("outbreak-disease") ? (
              <p
                id="outbreak-disease-error"
                className="mt-1 text-xs text-destructive"
              >
                {errorFor("outbreak-disease")}
              </p>
            ) : null}
          </div>
          <Field
            label="Geographic coverage"
            hint="needed to publish"
            value={form.geographic_area}
            onChange={(value) => field("geographic_area", value)}
            disabled={immutable}
          />
          <SelectField
            label="Region"
            value={form.region_id}
            onChange={(value) => {
              field("region_id", value);
              field("district_id", "");
            }}
            options={regions.map((value) => ({
              id: value.id,
              name: value.name,
            }))}
            empty="Select region"
            disabled={immutable}
          />
          <SelectField
            label="District"
            value={form.district_id}
            onChange={(value) => field("district_id", value)}
            options={districts.map((value) => ({
              id: value.id,
              name: value.name,
            }))}
            empty="Select district"
            disabled={immutable}
          />
          <Field
            label="Source organization"
            hint="needed to publish"
            value={form.source_organization}
            onChange={(value) => field("source_organization", value)}
            disabled={immutable}
          />
          <Field
            label="Source reference"
            hint="needed to publish"
            value={form.source_reference}
            onChange={(value) => field("source_reference", value)}
            disabled={immutable}
          />
          <Field
            label="Source HTTPS URL"
            hint="approved domains only, e.g. who.int, health.go.ug"
            value={form.source_url}
            onChange={(value) => field("source_url", value)}
            disabled={immutable}
          />
          <Field
            label="Start date"
            type="datetime-local"
            value={form.start_date}
            onChange={(value) => field("start_date", value)}
            disabled={immutable}
          />
          <Field
            label="Last update"
            type="datetime-local"
            value={form.last_update}
            onChange={(value) => field("last_update", value)}
            disabled={immutable}
          />
          <Field
            label="Effective at"
            hint="needed to publish"
            type="datetime-local"
            value={form.effective_at}
            onChange={(value) => field("effective_at", value)}
            disabled={immutable}
          />
          <Field
            label="Data as of"
            type="datetime-local"
            value={form.data_as_of}
            onChange={(value) => field("data_as_of", value)}
            disabled={immutable}
          />
          <Field
            label="Last verified"
            hint="needed to publish"
            type="datetime-local"
            value={form.last_verified_at}
            onChange={(value) => field("last_verified_at", value)}
            disabled={immutable}
          />
          <div>
            <Label>Visual tone</Label>
            <select
              className="mt-2 h-10 w-full rounded-md border bg-background px-3"
              value={form.visual_tone}
              disabled={immutable}
              onChange={(event) => field("visual_tone", event.target.value)}
            >
              {["neutral", "info", "warning", "critical", "success"].map(
                (value) => (
                  <option key={value}>{value}</option>
                ),
              )}
            </select>
          </div>
          <div className="md:col-span-2">
            <Label>Summary</Label>
            <Textarea
              className="mt-2"
              rows={5}
              value={form.summary}
              onChange={(event) => field("summary", event.target.value)}
              disabled={immutable}
            />
          </div>
          <div className="md:col-span-2">
            <Label>Change summary</Label>
            <Input
              className="mt-2"
              value={form.change_summary}
              onChange={(event) => field("change_summary", event.target.value)}
              placeholder="Explain the reason for this editorial change"
              disabled={immutable}
            />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Metrics</CardTitle>
            <Button
              variant="outline"
              size="sm"
              disabled={savingMetrics}
              onClick={openMetricDialog}
            >
              <Plus className="mr-2 h-4 w-4" />
              Metric
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            Metrics can be kept up to date at any time, even once the outbreak
            is published. To change a metric, remove it and add it again.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {metrics.length === 0 ? (
            <p className="text-sm text-muted-foreground">No metrics added.</p>
          ) : (
            metricPages.visible.map(({ metric, index }) => (
              <div
                key={metric._localId}
                className="flex items-start justify-between gap-4 rounded-md border p-3"
              >
                <div className="min-w-0 space-y-1">
                  <p className="font-medium">
                    {metric.label}{" "}
                    <span className="font-mono text-xs font-normal text-muted-foreground">
                      {metric.key}
                    </span>
                  </p>
                  <p className="text-sm">
                    {metric.value}
                    {metric.unit ? ` ${metric.unit}` : ""}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {metric.source_reference}
                    {metric.as_of
                      ? ` · as of ${new Date(metric.as_of).toLocaleString()}`
                      : ""}
                  </p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={savingMetrics}
                  onClick={() => void removeMetric(index)}
                >
                  Remove
                </Button>
              </div>
            ))
          )}
          <PaginationControls pagination={metricPages} label="Metrics" />
        </CardContent>
      </Card>
      <Dialog
        open={Boolean(metricDraft)}
        onOpenChange={(open) => {
          if (!open && !savingMetrics) setMetricDraft(null);
        }}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Add metric</DialogTitle>
            <DialogDescription>
              {item
                ? "The metric is saved as soon as you add it. Once the outbreak is published, adding a metric notifies everyone subscribed to outbreak alerts."
                : "The metric is saved with the outbreak."}
            </DialogDescription>
          </DialogHeader>
          {metricDraft ? (
            <div className="grid gap-3 md:grid-cols-2">
              <ChildField id="metric-label" label="Label" required error={metricError("metric-label")}>
                <Input
                  id="metric-label"
                  aria-required
                  aria-invalid={Boolean(metricError("metric-label"))}
                  placeholder="Confirmed cases"
                  value={metricDraft.label}
                  onChange={(event) => updateMetricDraft("label", event.target.value)}
                  autoFocus
                />
              </ChildField>
              <ChildField id="metric-key" label="Key" required error={metricError("metric-key")}>
                <Input
                  id="metric-key"
                  className="font-mono"
                  aria-required
                  aria-invalid={Boolean(metricError("metric-key"))}
                  placeholder="Generated from the label"
                  value={metricDraft.key}
                  onChange={(event) => {
                    // Clearing the key hands it back to the label.
                    setMetricKeyEdited(event.target.value !== "");
                    updateMetricDraft("key", event.target.value);
                  }}
                />
              </ChildField>
              <ChildField id="metric-value" label="Value" required error={metricError("metric-value")}>
                <Input
                  id="metric-value"
                  aria-required
                  aria-invalid={Boolean(metricError("metric-value"))}
                  placeholder="20"
                  value={metricDraft.value}
                  onChange={(event) => updateMetricDraft("value", event.target.value)}
                />
              </ChildField>
              <ChildField id="metric-unit" label="Unit">
                <Input
                  id="metric-unit"
                  placeholder="cases"
                  value={metricDraft.unit}
                  onChange={(event) => updateMetricDraft("unit", event.target.value)}
                />
              </ChildField>
              <ChildField
                id="metric-source_reference"
                label="Source"
                required
                error={metricError("metric-source_reference")}
              >
                <Input
                  id="metric-source_reference"
                  aria-required
                  aria-invalid={Boolean(metricError("metric-source_reference"))}
                  placeholder="WHO situation report 11"
                  value={metricDraft.source_reference}
                  onChange={(event) =>
                    updateMetricDraft("source_reference", event.target.value)
                  }
                />
              </ChildField>
              <ChildField id="metric-as_of" label="As of" required error={metricError("metric-as_of")}>
                <Input
                  id="metric-as_of"
                  type="datetime-local"
                  aria-required
                  aria-invalid={Boolean(metricError("metric-as_of"))}
                  value={toLocal(metricDraft.as_of)}
                  onChange={(event) =>
                    updateMetricDraft("as_of", iso(event.target.value) || "")
                  }
                />
              </ChildField>
            </div>
          ) : null}
          <DialogFooter>
            <Button
              variant="outline"
              disabled={savingMetrics}
              onClick={() => setMetricDraft(null)}
            >
              Cancel
            </Button>
            <Button disabled={savingMetrics} onClick={() => void addMetric()}>
              {savingMetrics ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Save className="mr-2 h-4 w-4" />
              )}
              {item ? "Save metric" : "Add metric"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {item ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Updates</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid items-end gap-2 md:grid-cols-[1fr_2fr_auto]">
                <ChildField
                  id="update-title"
                  label="Title"
                  required
                  error={updateError("update-title")}
                >
                  <Input
                    id="update-title"
                    placeholder="Update title"
                    aria-required
                    aria-invalid={Boolean(updateError("update-title"))}
                    value={updateDraft.title}
                    onChange={(event) =>
                      setUpdateDraft((current) => ({
                        ...current,
                        title: event.target.value,
                      }))
                    }
                  />
                </ChildField>
                <ChildField id="update-summary" label="Summary">
                  <Input
                    id="update-summary"
                    placeholder="Verified update summary"
                    value={updateDraft.summary}
                    onChange={(event) =>
                      setUpdateDraft((current) => ({
                        ...current,
                        summary: event.target.value,
                      }))
                    }
                  />
                </ChildField>
                <Button onClick={() => void addUpdate()}>
                  Add draft
                </Button>
              </div>
              <ChildContentWorkflow
                outbreakId={item.id!}
                kind="update"
                pageSize={DEFAULT_PAGE_SIZE}
                items={updates}
                empty="No updates yet."
                onChanged={hydrate}
                currentUserId={currentUserId}
                parentPublished={isPublished}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Typed resources</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid items-end gap-2 md:grid-cols-2">
                <ResourceFields
                  className="contents"
                  draft={resourceDraft}
                  onChange={setResourceDraft}
                  documentKinds={documentKinds}
                  reports={reports}
                  errorFor={resourceError}
                />
                <Button
                  className="md:col-span-2"
                  disabled={resourcesLocked}
                  onClick={() => void addResource()}
                >
                  Add resource draft
                </Button>
                {resourcesLocked ? (
                  <p className="text-xs text-muted-foreground md:col-span-2">
                    Resources can&apos;t be added to an outbreak that is {item.status}.
                  </p>
                ) : null}
              </div>
              <ChildContentWorkflow
                outbreakId={item.id!}
                kind="resource"
                items={resources}
                empty="No resources yet."
                onChanged={hydrate}
                currentUserId={currentUserId}
                parentPublished={isPublished}
                documentKinds={documentKinds}
                reports={reports}
                pageSize={DEFAULT_PAGE_SIZE}
                linkTitles={Object.fromEntries([
                  ...guidelines.map((value) => [`/public/guidelines/${value.id}`, value.title || ""]),
                  ...reports.map((value) => [`/situation-reports/${value.id}`, value.title || ""]),
                ])}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Related situation reports</CardTitle>
            </CardHeader>
            <CardContent>
              <ChildRows
                items={reports.map((value) => ({
                  id: value.id!,
                  title: value.title || "Untitled report",
                  status: value.status || "draft",
                  detail: value.publication_date
                    ? new Date(value.publication_date).toLocaleDateString()
                    : "",
                }))}
                empty="No situation reports are linked."
                label="Situation reports"
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Review and audit history</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex gap-2">
                <Textarea
                  maxLength={4000}
                  value={reviewComment}
                  onChange={(event) => setReviewComment(event.target.value)}
                  placeholder="Add an auditable reviewer comment"
                />
                <Button
                  variant="outline"
                  disabled={!reviewComment.trim()}
                  onClick={() => void addReviewComment()}
                >
                  Add comment
                </Button>
              </div>
              <ChildRows
                items={audit.map((value) => ({
                  id: value.id,
                  title: value.action,
                  status: new Date(value.created_at).toLocaleString(),
                  detail:
                    typeof value.metadata?.comment === "string"
                      ? value.metadata.comment
                      : `Actor ${value.actor_id}`,
                }))}
                empty="No audit events recorded."
                label="Audit history"
              />
            </CardContent>
          </Card>
        </>
      ) : null}
      {item &&
      ["published", "active", "monitoring", "contained", "closed"].includes(
        item.status || "",
      ) ? (
        <CampaignDraftBuilder
          contentKey={`outbreak:${item.id}:${item.lock_version}`}
          kinds={[
            { value: "alert", label: "Outbreak alert" },
            { value: "update", label: "Outbreak update" },
            { value: "status_change", label: "Status change" },
            { value: "closure", label: "Closure" },
          ]}
          create={(input) => outbreaksService.createCampaign(item.id!, input)}
        />
      ) : null}
    </div>
  );

  function updateMetricDraft(key: keyof MetricDraft, value: string) {
    setMetricDraft((current) => {
      if (!current) return current;
      const next = { ...current, [key]: value };
      if (key === "label" && !metricKeyEdited) {
        next.key = metricKeyFromLabel(value, metrics);
      }
      return next;
    });
  }
}

function ChildField({
  id,
  label,
  required,
  error,
  children,
}: {
  id: string;
  label: string;
  required?: boolean;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1">
      <Label htmlFor={id}>
        {label}
        {required ? (
          <>
            <RequiredMark />
            <span className="sr-only">(required)</span>
          </>
        ) : null}
      </Label>
      {children}
      {error ? (
        <p id={`${id}-error`} className="text-xs text-destructive">
          {error}
        </p>
      ) : null}
    </div>
  );
}

function Field({
  id,
  label,
  value,
  onChange,
  type = "text",
  required,
  hint,
  error,
  disabled,
}: {
  id?: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  required?: boolean;
  hint?: string;
  error?: string;
  disabled?: boolean;
}) {
  return (
    <div>
      <Label htmlFor={id}>
        {label}
        {required ? (
          <>
            <RequiredMark />
            <span className="sr-only">(required)</span>
          </>
        ) : null}
        {hint ? (
          <span className="text-xs font-normal text-muted-foreground">
            ({hint})
          </span>
        ) : null}
      </Label>
      <Input
        id={id}
        className="mt-2"
        type={type}
        value={value}
        required={required}
        disabled={disabled}
        aria-invalid={Boolean(error)}
        aria-describedby={error && id ? `${id}-error` : undefined}
        onChange={(event) => onChange(event.target.value)}
      />
      {error ? (
        <p
          id={id ? `${id}-error` : undefined}
          className="mt-1 text-xs text-destructive"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}
function SelectField({
  label,
  value,
  onChange,
  options,
  empty,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ id: string; name: string }>;
  empty: string;
  disabled?: boolean;
}) {
  return (
    <div>
      <Label>{label}</Label>
      <select
        className="mt-2 h-10 w-full rounded-md border bg-background px-3"
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{empty}</option>
        {options.map((option) => (
          <option key={option.id} value={option.id}>
            {option.name}
          </option>
        ))}
      </select>
    </div>
  );
}
function ChildRows({
  items,
  empty,
  label,
}: {
  items: Array<{ id: string; title: string; status: string; detail?: string }>;
  empty: string;
  label: string;
}) {
  const pagination = usePagination(items, DEFAULT_PAGE_SIZE);
  return items.length ? (
    <div className="space-y-2">
      {pagination.visible.map((item) => (
        <div
          key={item.id}
          className="flex items-center justify-between rounded-md border p-3"
        >
          <div>
            <div className="font-medium">{item.title}</div>
            <div className="text-xs text-muted-foreground">{item.detail}</div>
          </div>
          <Badge variant="outline">{item.status}</Badge>
        </div>
      ))}
      <PaginationControls pagination={pagination} label={label} />
    </div>
  ) : (
    <div className="flex items-center gap-2 text-sm text-muted-foreground">
      <AlertCircle className="h-4 w-4" />
      {empty}
    </div>
  );
}
function iso(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}
function toLocal(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
}
function conflictMessage(value: unknown) {
  const message = value instanceof Error ? value.message : "Operation failed";
  return /conflict|modified|lock/i.test(message)
    ? "Another editor changed this record. Reload before retrying."
    : message;
}
