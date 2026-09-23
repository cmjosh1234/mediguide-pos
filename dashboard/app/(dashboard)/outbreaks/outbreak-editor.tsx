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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MultiSelect } from "@/components/ui/multi-select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { getCurrentUser } from "@/lib/backend-client";
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
import type { DistrictsResponse, RegionsResponse } from "@/types/backend-types";
import { RequiredMark } from "./required-field";
import { CampaignDraftBuilder } from "./campaign-draft-builder";
import { ChildContentWorkflow } from "./child-content-workflow";
import { OutbreakDocumentsWorkspace } from "./outbreak-documents-workspace";

type MetricDraft = {
  key: string;
  label: string;
  value: string;
  numeric_value?: number;
  unit: string;
  as_of: string;
  source_reference: string;
  sort_order: number;
};

const emptyMetric = (): MetricDraft => ({
  key: "",
  label: "",
  value: "",
  unit: "",
  as_of: new Date().toISOString(),
  source_reference: "",
  sort_order: 1,
});

type Problem = { id: string; label: string; message: string };

// Mirrors the backend rules that reject a draft save (outbreak_validation.go).
const METRIC_KEY_PATTERN = /^[a-z][a-z0-9_]{1,63}$/;

function findProblems(
  form: { title: string; disease_id: string },
  metrics: MetricDraft[],
): Problem[] {
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
  metrics.forEach((metric, index) => {
    const name = `Metric ${index + 1}`;
    const key = metric.key.trim().toLowerCase();
    if (!key) {
      problems.push({ id: `metric-${index}-key`, label: name, message: `${name}: key is required.` });
    } else if (!METRIC_KEY_PATTERN.test(key)) {
      problems.push({
        id: `metric-${index}-key`,
        label: name,
        message: `${name}: key must be lowercase letters, numbers or underscores, starting with a letter.`,
      });
    }
    if (!metric.label.trim()) {
      problems.push({ id: `metric-${index}-label`, label: name, message: `${name}: label is required.` });
    }
    if (!metric.value.trim()) {
      problems.push({ id: `metric-${index}-value`, label: name, message: `${name}: value is required.` });
    }
    if (!metric.source_reference.trim()) {
      problems.push({
        id: `metric-${index}-source_reference`,
        label: name,
        message: `${name}: source is required.`,
      });
    }
  });
  return problems;
}

type ResourceDraft = {
  title: string;
  resource_type: string;
  url: string;
  asset_url: string;
};

function findUpdateProblems(draft: { title: string }): Problem[] {
  return draft.title.trim()
    ? []
    : [{ id: "update-title", label: "Title", message: "Title is required." }];
}

function findResourceProblems(draft: ResourceDraft): Problem[] {
  const problems: Problem[] = [];
  if (!draft.title.trim()) {
    problems.push({ id: "resource-title", label: "Title", message: "Title is required." });
  }
  const type = draft.resource_type;
  if (type === "managed_document" || type === "downloadable_asset") {
    if (!draft.asset_url.trim()) {
      problems.push({ id: "resource-asset", label: "Asset path", message: "Asset path or URL is required." });
    }
  } else if (!draft.url.trim()) {
    const [label, message] =
      type === "guideline"
        ? ["Published guideline", "Select a published guideline."]
        : type === "situation_report"
          ? ["Published situation report", "Select a published situation report."]
          : ["URL", "A route or HTTPS URL is required."];
    problems.push({ id: "resource-url", label, message });
  }
  return problems;
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

export function OutbreakEditor({
  id,
  initialDocumentId,
}: {
  id?: string;
  initialDocumentId?: string;
}) {
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
  const [updateDraft, setUpdateDraft] = React.useState({
    title: "",
    summary: "",
  });
  const [resourceDraft, setResourceDraft] = React.useState({
    title: "",
    description: "",
    issuing_organization: "",
    resource_type: "guideline",
    url: "",
    asset_url: "",
  });
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
      setMetrics((record.metrics || []) as MetricDraft[]);
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
    () => findProblems(form, metrics),
    [form, metrics],
  );
  const visibleProblems = attempted ? problems : [];
  const errorFor = (id: string) =>
    visibleProblems.find((problem) => problem.id === id)?.message;

  function focusField(id: string) {
    const element = document.getElementById(id);
    if (!element) return;
    element.scrollIntoView({ block: "center", behavior: "smooth" });
    element.focus({ preventScroll: true });
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
        metrics,
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
    } catch (value) {
      showToast.error("Unable to save", conflictMessage(value), {
        richColors: true,
      });
    } finally {
      setSaving(false);
    }
  }

  async function saveMetrics() {
    if (!item) return;
    const metricProblems = problems.filter((problem) =>
      problem.id.startsWith("metric-"),
    );
    if (metricProblems.length > 0) {
      setAttempted(true);
      rejectIncomplete(metricProblems);
      return;
    }
    setSavingMetrics(true);
    try {
      const saved = await outbreaksService.updateMetrics(item.id!, {
        metrics,
        lock_version: item.lock_version,
      });
      showToast.success("Metrics saved", "The outbreak's metrics were updated.");
      setItem(saved);
      setMetrics((saved.metrics || []) as MetricDraft[]);
    } catch (value) {
      showToast.error("Unable to save metrics", conflictMessage(value), {
        richColors: true,
      });
    } finally {
      setSavingMetrics(false);
    }
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
        ...resourceDraft,
        sort_order: resources.length + 1,
      });
      setResources((current) => [...current, created]);
      setResourceDraft({
        title: "",
        description: "",
        issuing_organization: "",
        resource_type: "guideline",
        url: "",
        asset_url: "",
      });
      setResourceAttempted(false);
      showToast.success(
        "Resource draft created",
        "The link passed the backend trust boundary.",
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
                <Link href={`/api/public/outbreaks/${item.id}`} target="_blank">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Preview public API
                </Link>
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
            <div className="flex items-center gap-2">
              {item ? (
                <Button
                  variant="default"
                  size="sm"
                  disabled={savingMetrics}
                  onClick={() => void saveMetrics()}
                >
                  {savingMetrics ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  Save metrics
                </Button>
              ) : null}
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  setMetrics((current) => [
                    ...current,
                    { ...emptyMetric(), sort_order: current.length + 1 },
                  ])
                }
              >
                <Plus className="mr-2 h-4 w-4" />
                Metric
              </Button>
            </div>
          </div>
          <p className="text-sm text-muted-foreground">
            Metrics can be kept up to date at any time, even once the outbreak
            is published.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {metrics.length > 0 ? (
            <p className="text-sm text-muted-foreground">
              <RequiredMark /> Key, label, value and source are required for
              each metric.
            </p>
          ) : null}
          {metrics.length === 0 ? (
            <p className="text-sm text-muted-foreground">No metrics added.</p>
          ) : (
            metrics.map((metric, index) => (
              <div
                key={index}
                className="grid gap-2 rounded-md border p-3 md:grid-cols-4"
              >
                <Input
                  id={`metric-${index}-key`}
                  aria-label="Metric key"
                  aria-required
                  aria-invalid={Boolean(errorFor(`metric-${index}-key`))}
                  placeholder="confirmed_cases"
                  value={metric.key}
                  onChange={(event) =>
                    updateMetric(index, "key", event.target.value)
                  }
                />
                <Input
                  id={`metric-${index}-label`}
                  aria-label="Metric label"
                  aria-required
                  aria-invalid={Boolean(errorFor(`metric-${index}-label`))}
                  placeholder="Confirmed cases"
                  value={metric.label}
                  onChange={(event) =>
                    updateMetric(index, "label", event.target.value)
                  }
                />
                <Input
                  id={`metric-${index}-value`}
                  aria-label="Metric value"
                  aria-required
                  aria-invalid={Boolean(errorFor(`metric-${index}-value`))}
                  placeholder="20"
                  value={metric.value}
                  onChange={(event) =>
                    updateMetric(index, "value", event.target.value)
                  }
                />
                <Input
                  aria-label="Metric unit"
                  placeholder="cases"
                  value={metric.unit}
                  onChange={(event) =>
                    updateMetric(index, "unit", event.target.value)
                  }
                />
                <Input
                  className="md:col-span-2"
                  id={`metric-${index}-source_reference`}
                  aria-label="Metric source"
                  aria-required
                  aria-invalid={Boolean(errorFor(`metric-${index}-source_reference`))}
                  placeholder="WHO situation report 11"
                  value={metric.source_reference}
                  onChange={(event) =>
                    updateMetric(index, "source_reference", event.target.value)
                  }
                />
                <Input
                  type="datetime-local"
                  aria-label="Metric as of"
                  value={toLocal(metric.as_of)}
                  onChange={(event) =>
                    updateMetric(index, "as_of", iso(event.target.value) || "")
                  }
                />
                <Button
                  variant="destructive"
                  onClick={() =>
                    setMetrics((current) =>
                      current.filter((_, position) => position !== index),
                    )
                  }
                >
                  Remove
                </Button>
              </div>
            ))
          )}
        </CardContent>
      </Card>
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
                <Button disabled={immutable} onClick={() => void addUpdate()}>
                  Add draft
                </Button>
              </div>
              <ChildContentWorkflow
                outbreakId={item.id!}
                kind="update"
                items={updates}
                empty="No updates yet."
                onChanged={hydrate}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Typed resources</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid items-end gap-2 md:grid-cols-2">
                <ChildField
                  id="resource-title"
                  label="Title"
                  required
                  error={resourceError("resource-title")}
                >
                  <Input
                    id="resource-title"
                    placeholder="Resource title"
                    aria-required
                    aria-invalid={Boolean(resourceError("resource-title"))}
                    value={resourceDraft.title}
                    onChange={(event) =>
                      setResourceDraft((current) => ({
                        ...current,
                        title: event.target.value,
                      }))
                    }
                  />
                </ChildField>
                <ChildField id="resource-type" label="Type">
                  <select
                    id="resource-type"
                    className="h-10 w-full rounded-md border bg-background px-3"
                    value={resourceDraft.resource_type}
                    onChange={(event) =>
                      setResourceDraft((current) => ({
                        ...current,
                        resource_type: event.target.value,
                        url: "",
                        asset_url: "",
                      }))
                    }
                  >
                    {[
                      "guideline",
                      "situation_report",
                      "internal_route",
                      "approved_external_url",
                      "managed_document",
                      "downloadable_asset",
                    ].map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                </ChildField>
                <ChildField id="resource-issuer" label="Issuing organization">
                  <Input
                    id="resource-issuer"
                    placeholder="Issuing organization"
                    value={resourceDraft.issuing_organization}
                    onChange={(event) =>
                      setResourceDraft((current) => ({
                        ...current,
                        issuing_organization: event.target.value,
                      }))
                    }
                  />
                </ChildField>
                <ChildField id="resource-description" label="Description">
                  <Input
                    id="resource-description"
                    placeholder="Short public description"
                    value={resourceDraft.description}
                    onChange={(event) =>
                      setResourceDraft((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                  />
                </ChildField>
                {["managed_document", "downloadable_asset"].includes(
                  resourceDraft.resource_type,
                ) ? (
                  <>
                    <ChildField id="resource-managed" label="Resource URL">
                      <Input
                        id="resource-managed"
                        disabled
                        value="Backend-managed asset only"
                      />
                    </ChildField>
                    <ChildField
                      id="resource-asset"
                      label="Asset path"
                      required
                      error={resourceError("resource-asset")}
                    >
                      <Input
                        id="resource-asset"
                        placeholder="Backend-managed report asset path or approved HTTPS URL"
                        aria-required
                        aria-invalid={Boolean(resourceError("resource-asset"))}
                        value={resourceDraft.asset_url}
                        onChange={(event) =>
                          setResourceDraft((current) => ({
                            ...current,
                            asset_url: event.target.value,
                          }))
                        }
                      />
                    </ChildField>
                  </>
                ) : (
                  <ChildField
                    id="resource-url"
                    label={
                      resourceDraft.resource_type === "guideline"
                        ? "Published guideline"
                        : resourceDraft.resource_type === "situation_report"
                          ? "Published situation report"
                          : "Route or URL"
                    }
                    required
                    error={resourceError("resource-url")}
                  >
                    {resourceDraft.resource_type === "guideline" ||
                    resourceDraft.resource_type === "situation_report" ? (
                      <select
                        id="resource-url"
                        aria-required
                        aria-invalid={Boolean(resourceError("resource-url"))}
                        className="h-10 w-full rounded-md border bg-background px-3 aria-invalid:border-destructive aria-invalid:ring-[3px] aria-invalid:ring-destructive/20"
                        value={resourceDraft.url}
                        onChange={(event) =>
                          setResourceDraft((current) => ({
                            ...current,
                            url: event.target.value,
                          }))
                        }
                      >
                        {resourceDraft.resource_type === "guideline" ? (
                          <>
                            <option value="">Select a published guideline</option>
                            {guidelines.map((value) => (
                              <option
                                key={value.id}
                                value={`/public/guidelines/${value.id}`}
                              >
                                {value.title}
                              </option>
                            ))}
                          </>
                        ) : (
                          <>
                            <option value="">
                              Select a published situation report
                            </option>
                            {reports
                              .filter((value) => value.status === "published")
                              .map((value) => (
                                <option
                                  key={value.id}
                                  value={`/situation-reports/${value.id}`}
                                >
                                  {value.title}
                                </option>
                              ))}
                          </>
                        )}
                      </select>
                    ) : (
                      <Input
                        id="resource-url"
                        placeholder="Allowlisted internal route or approved HTTPS URL"
                        aria-required
                        aria-invalid={Boolean(resourceError("resource-url"))}
                        value={resourceDraft.url}
                        onChange={(event) =>
                          setResourceDraft((current) => ({
                            ...current,
                            url: event.target.value,
                          }))
                        }
                      />
                    )}
                  </ChildField>
                )}
                <Button
                  className="md:col-span-2"
                  disabled={immutable}
                  onClick={() => void addResource()}
                >
                  Add resource draft
                </Button>
              </div>
              <ChildContentWorkflow
                outbreakId={item.id!}
                kind="resource"
                items={resources}
                empty="No resources yet."
                onChanged={hydrate}
              />
            </CardContent>
          </Card>
            <OutbreakDocumentsWorkspace
              outbreakId={item.id!}
              initialDocumentId={initialDocumentId}
            />
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

  function updateMetric(index: number, key: keyof MetricDraft, value: string) {
    setMetrics((current) =>
      current.map((metric, position) =>
        position === index ? { ...metric, [key]: value } : metric,
      ),
    );
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
}: {
  items: Array<{ id: string; title: string; status: string; detail?: string }>;
  empty: string;
}) {
  return items.length ? (
    <div className="space-y-2">
      {items.map((item) => (
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
