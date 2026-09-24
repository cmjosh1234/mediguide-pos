"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Image from "next/image";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import {
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  Check,
  ChevronLeft,
  ChevronRight,
  CircleCheck,
  ExternalLink,
  FileText,
  GitMerge,
  Monitor,
  MoreHorizontal,
  Pencil,
  Save,
  Scissors,
  Send,
  Settings2,
  Smartphone,
  Trash2,
  X,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingState } from "@/components/ui/loading-state";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { showToast } from "@/lib/toast";
import { GuidelineBulkReviewPanel } from "@/components/guidelines/guideline-bulk-review-panel";
import { GuidelineCompletenessReport } from "@/components/guidelines/guideline-completeness-report";
import { GuidelineReviewIssuesPopover } from "@/components/guidelines/guideline-review-issues";
import {
  guidelineDocumentsQueryKey,
  GuidelineBlockType,
  GuidelineAssetRecord,
  GuidelineContentBlockRecord,
  GuidelineDocumentsService,
  GuidelineReviewIssue,
  GuidelineSectionRecord,
} from "@/services/guideline-documents.service";

const blockTypes: GuidelineBlockType[] = [
  "heading",
  "paragraph",
  "ordered_list",
  "unordered_list",
  "table",
  "figure",
  "recommendation",
  "warning",
  "caution",
  "key_point",
  "contraindication",
  "dosage",
  "evidence",
  "definition",
  "procedure",
  "clinical_note",
  "referral_criteria",
  "algorithm_reference",
  "algorithm",
  "reference",
  "page_break",
  "unknown",
];

type BlockReviewFilter = "high-risk" | "pending-high-risk" | "all";

function blockText(block: GuidelineContentBlockRecord) {
  const content = block.content;
  if (typeof content.text === "string") return content.text;
  if (typeof content.content === "string") return content.content;
  if (typeof content.citation === "string") return content.citation;
  if (Array.isArray(content.items))
    return content.items
      .filter((item): item is string => typeof item === "string")
      .join("\n");
  if (typeof content.title === "string") return content.title;
  if (typeof content.caption === "string") return content.caption;
  if (typeof content.alternative_text === "string")
    return content.alternative_text;
  return "";
}

function BlockPreview({
  block,
  asset,
}: {
  block: GuidelineContentBlockRecord;
  asset?: GuidelineAssetRecord;
}) {
  const content = block.content;
  const text = blockText(block);
  if (block.type === "heading") {
    const level = Math.min(6, Math.max(1, Number(content.level) || 2));
    const Heading = `h${level}` as keyof React.JSX.IntrinsicElements;
    return <Heading className="font-semibold">{text}</Heading>;
  }
  if (block.type === "ordered_list" || block.type === "unordered_list") {
    const List = block.type === "ordered_list" ? "ol" : "ul";
    return (
      <List className="ml-5 list-outside list-disc space-y-1">
        {((content.items as unknown[]) || []).map((item, index) => (
          <li key={index}>{String(item)}</li>
        ))}
      </List>
    );
  }
  if (block.type === "table") {
    const columns = Array.isArray(content.columns) ? content.columns : [];
    const rows = Array.isArray(content.rows) ? content.rows : [];
    return (
      <div className="min-w-0 overflow-hidden rounded-md border">
        <table className="w-full table-fixed border-collapse text-xs">
          <caption className="border-b bg-muted/50 p-2 text-left font-semibold">
            {text || "Clinical table"}
          </caption>
          <thead>
            <tr>
              {columns.map((column, index) => (
                <th
                  className="break-words border-b border-r bg-muted p-2 text-left align-top [overflow-wrap:anywhere] last:border-r-0"
                  key={index}
                >
                  {String(column)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => (
              <tr key={rowIndex}>
                {(Array.isArray(row) ? row : []).map((cell, cellIndex) => (
                  <td
                    className="break-words border-b border-r p-2 align-top [overflow-wrap:anywhere] last:border-r-0"
                    key={cellIndex}
                  >
                    {String(cell)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  if (["recommendation", "warning", "key_point"].includes(block.type)) {
    return (
      <div
        className={
          block.type === "warning"
            ? "rounded-md border border-amber-300 bg-amber-50 p-3 text-amber-950"
            : "rounded-md border border-blue-200 bg-blue-50 p-3 text-blue-950"
        }
      >
        <div className="text-xs font-semibold uppercase">
          {String(content.title || block.type.replace("_", " "))}
        </div>
        <p className="mt-1 whitespace-pre-wrap text-sm">{text}</p>
      </div>
    );
  }
  if (block.type === "figure")
    return <FigureBlockPreview block={block} asset={asset} />;
  if (block.type === "algorithm")
    return (
      <div className="rounded-md border border-dashed p-6 text-center text-sm">
        Reviewed structured algorithm ·{" "}
        {Array.isArray(content.nodes) ? content.nodes.length : 0} nodes
      </div>
    );
  if (block.type === "page_break")
    return (
      <div className="border-t border-dashed pt-2 text-xs text-muted-foreground">
        Page {String(content.page || "")}
      </div>
    );
  return (
    <p className="whitespace-pre-wrap text-sm leading-6">
      {text || JSON.stringify(content)}
    </p>
  );
}

function FigureBlockPreview({
  block,
  asset,
}: {
  block: GuidelineContentBlockRecord;
  asset?: GuidelineAssetRecord;
}) {
  const assetId =
    typeof block.content.asset_id === "string" ? block.content.asset_id : "";
  const [url, setUrl] = React.useState<string | null>(null);
  React.useEffect(() => {
    if (!assetId) return;
    let active = true;
    GuidelineDocumentsService.getReviewAsset(block.version_id, assetId)
      .then((blob) => {
        if (!active) return;
        setUrl(URL.createObjectURL(blob));
      })
      .catch(() => setUrl(null));
    return () => {
      active = false;
    };
  }, [assetId, block.version_id]);
  React.useEffect(
    () => () => {
      if (url) URL.revokeObjectURL(url);
    },
    [url],
  );
  const caption =
    (typeof block.content.caption === "string" ? block.content.caption : "") ||
    asset?.caption ||
    "";
  const alternativeText =
    (typeof block.content.alternative_text === "string"
      ? block.content.alternative_text
      : "") ||
    asset?.alternative_text ||
    caption;
  const source =
    (typeof block.content.source === "string" ? block.content.source : "") ||
    asset?.source ||
    "";
  const attribution =
    (typeof block.content.attribution === "string"
      ? block.content.attribution
      : "") ||
    asset?.attribution ||
    "";
  const license =
    (typeof block.content.license === "string" ? block.content.license : "") ||
    asset?.license ||
    "";
  return (
    <figure className="space-y-2 rounded-md border p-2">
      {url ? (
        <Image
          unoptimized
          width={800}
          height={600}
          src={url}
          alt={
            alternativeText ||
            "Extracted guideline figure pending alternative text"
          }
          className="mx-auto h-auto max-h-96 w-auto max-w-full object-contain"
        />
      ) : (
        <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
          Figure asset unavailable
        </div>
      )}
      <figcaption className="text-xs text-muted-foreground">
        {caption || "Caption required before review"}
      </figcaption>
      <dl className="grid gap-1 border-t pt-2 text-xs">
        <div>
          <dt className="inline font-medium">Alternative text: </dt>
          <dd className="inline">
            {alternativeText || "Missing — add it before approval"}
          </dd>
        </div>
        {source && (
          <div>
            <dt className="inline font-medium">Source: </dt>
            <dd className="inline">{source}</dd>
          </div>
        )}
        {attribution && (
          <div>
            <dt className="inline font-medium">Attribution: </dt>
            <dd className="inline">{attribution}</dd>
          </div>
        )}
        {license && (
          <div>
            <dt className="inline font-medium">License: </dt>
            <dd className="inline">{license}</dd>
          </div>
        )}
      </dl>
    </figure>
  );
}


type ReviewTab = "review" | "bulk" | "completeness";
type CompactPanel = "sections" | "source" | "block";

function isEditableTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) ||
    Boolean(target.closest("[role=dialog],[role=menu],[role=listbox]"))
  );
}

export default function GuidelineReviewPage() {
  const { id, versionId } = useParams<{ id: string; versionId: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const pendingQueueInitialized = React.useRef(false);
  const cameFromRegeneration = searchParams.get("focus") === "pending-high-risk";
  const [tab, setTab] = React.useState<ReviewTab>("review");
  const [compactPanel, setCompactPanel] = React.useState<CompactPanel>("block");
  const [centerView, setCenterView] = React.useState<"source" | "preview">(
    "source",
  );
  const [sectionDialogOpen, setSectionDialogOpen] = React.useState(false);
  const [selectedSectionId, setSelectedSectionId] = React.useState<
    string | null
  >(null);
  const [selectedBlockId, setSelectedBlockId] = React.useState<string | null>(
    null,
  );
  const [pdfUrl, setPdfUrl] = React.useState<string | null>(null);
  const [pdfPage, setPdfPage] = React.useState(1);
  const [previewMode, setPreviewMode] = React.useState<"web" | "mobile">("web");
  const [blockReviewFilter, setBlockReviewFilter] =
    React.useState<BlockReviewFilter>(() =>
      cameFromRegeneration ? "pending-high-risk" : "high-risk",
    );
  const [sectionDraft, setSectionDraft] = React.useState({
    title: "",
    slug: "",
    level: 1,
  });
  const [blockType, setBlockType] =
    React.useState<GuidelineBlockType>("paragraph");
  const [blockJSON, setBlockJSON] = React.useState("");

  const workspaceQuery = useQuery({
    queryKey: [...guidelineDocumentsQueryKey, versionId, "review"],
    queryFn: () => GuidelineDocumentsService.getReviewWorkspace(versionId),
    enabled: Boolean(versionId),
  });

  React.useEffect(() => {
    let active = true;
    GuidelineDocumentsService.getOriginalPdf(versionId)
      .then((blob) => {
        if (!active) return;
        const url = URL.createObjectURL(blob);
        setPdfUrl((previous) => {
          if (previous) URL.revokeObjectURL(previous);
          return url;
        });
      })
      .catch(() => setPdfUrl(null));
    return () => {
      active = false;
    };
  }, [versionId]);

  React.useEffect(
    () => () => {
      if (pdfUrl) URL.revokeObjectURL(pdfUrl);
    },
    [pdfUrl],
  );

  const workspace = workspaceQuery.data;
  const sections = React.useMemo(
    () => workspace?.sections || [],
    [workspace?.sections],
  );
  const blocks = React.useMemo(
    () => workspace?.blocks || [],
    [workspace?.blocks],
  );
  // Blocks in reading order, so "next" always means further down the document.
  const orderedBlocks = React.useMemo(() => {
    const sectionIndex = new Map(
      sections.map((section, index) => [section.id, index]),
    );
    const position = (block: GuidelineContentBlockRecord) =>
      sectionIndex.get(block.section_id ?? "") ?? sections.length;
    return [...blocks].sort(
      (left, right) =>
        position(left) - position(right) || left.sort_order - right.sort_order,
    );
  }, [blocks, sections]);
  const individualReviewBlockTypes = React.useMemo(
    () =>
      new Set([
        ...(workspace?.block_review_policy?.high_risk_types || []),
        ...(workspace?.block_review_policy?.conditional_risk_types || []),
      ]),
    [
      workspace?.block_review_policy?.conditional_risk_types,
      workspace?.block_review_policy?.high_risk_types,
    ],
  );
  const individualReviewBlocks = React.useMemo(
    () =>
      orderedBlocks.filter((block) =>
        individualReviewBlockTypes.has(block.type),
      ),
    [orderedBlocks, individualReviewBlockTypes],
  );
  const pendingIndividualReviewBlocks = React.useMemo(
    () =>
      individualReviewBlocks.filter(
        (block) => block.review_status !== "reviewed",
      ),
    [individualReviewBlocks],
  );
  const sectionReviewCounts = React.useMemo(() => {
    const counts = new Map<string, { total: number; pending: number }>();
    for (const block of individualReviewBlocks) {
      if (!block.section_id) continue;
      const current = counts.get(block.section_id) || { total: 0, pending: 0 };
      current.total += 1;
      if (block.review_status !== "reviewed") current.pending += 1;
      counts.set(block.section_id, current);
    }
    return counts;
  }, [individualReviewBlocks]);

  const selectedSection =
    sections.find((section) => section.id === selectedSectionId) || sections[0];
  const sectionBlocks = orderedBlocks.filter(
    (block) => block.section_id === selectedSection?.id,
  );
  const visibleSectionBlocks = sectionBlocks.filter((block) => {
    if (blockReviewFilter === "all") return true;
    if (!individualReviewBlockTypes.has(block.type)) return false;
    return (
      blockReviewFilter !== "pending-high-risk" ||
      block.review_status !== "reviewed"
    );
  });
  const selectedBlock =
    visibleSectionBlocks.find((block) => block.id === selectedBlockId) ||
    visibleSectionBlocks[0];
  const blockDirty = Boolean(
    selectedBlock &&
      (blockType !== selectedBlock.type ||
        blockJSON !== JSON.stringify(selectedBlock.content, null, 2)),
  );

  React.useEffect(() => {
    if (!selectedSectionId && sections[0]) setSelectedSectionId(sections[0].id);
  }, [sections, selectedSectionId]);

  const selectBlock = React.useCallback(
    (block: GuidelineContentBlockRecord) => {
      if (block.section_id) setSelectedSectionId(block.section_id);
      setSelectedBlockId(block.id);
    },
    [],
  );

  const nextPendingAfter = React.useCallback(
    (blockId: string | undefined) => {
      const candidates = pendingIndividualReviewBlocks.filter(
        (block) => block.id !== blockId,
      );
      if (!blockId) return candidates[0];
      const position = orderedBlocks.findIndex((block) => block.id === blockId);
      return (
        candidates.find(
          (block) =>
            orderedBlocks.findIndex((item) => item.id === block.id) > position,
        ) || candidates[0]
      );
    },
    [orderedBlocks, pendingIndividualReviewBlocks],
  );

  const startReview = React.useCallback(() => {
    setTab("review");
    setBlockReviewFilter("pending-high-risk");
    setCompactPanel("block");
    const next = pendingIndividualReviewBlocks[0];
    if (next) selectBlock(next);
  }, [pendingIndividualReviewBlocks, selectBlock]);

  const focusReviewIssue = React.useCallback(
    (issue: GuidelineReviewIssue) => {
      setTab("review");
      const affectedBlock = issue.block_id
        ? blocks.find((block) => block.id === issue.block_id)
        : issue.asset_id
          ? blocks.find(
              (block) =>
                block.type === "figure" &&
                block.content.asset_id === issue.asset_id,
            )
          : undefined;
      if (affectedBlock) {
        // A previously reviewed figure can still have a draft asset after its
        // metadata changes, so reveal it even when the pending filter is active.
        setBlockReviewFilter("all");
        setCompactPanel("block");
        selectBlock(affectedBlock);
        return;
      }
      if (issue.section_id) {
        setSelectedSectionId(issue.section_id);
        setCompactPanel("sections");
      }
    },
    [blocks, selectBlock],
  );

  // Open on the first block that still needs review instead of the first
  // section, which is usually front matter with nothing to approve.
  React.useEffect(() => {
    if (!workspace || pendingQueueInitialized.current) return;
    pendingQueueInitialized.current = true;
    const firstPending = pendingIndividualReviewBlocks[0];
    if (!firstPending) return;
    setBlockReviewFilter("pending-high-risk");
    selectBlock(firstPending);
  }, [workspace, pendingIndividualReviewBlocks, selectBlock]);

  React.useEffect(() => {
    if (!selectedSection) return;
    setSectionDraft({
      title: selectedSection.title,
      slug: selectedSection.slug,
      level: selectedSection.level,
    });
  }, [selectedSection]);

  React.useEffect(() => {
    if (!selectedBlock) return;
    setSelectedBlockId(selectedBlock.id);
    setBlockType(selectedBlock.type);
    setBlockJSON(JSON.stringify(selectedBlock.content, null, 2));
    setPdfPage(selectedBlock.page_start || selectedSection?.page_start || 1);
  }, [selectedBlock, selectedSection?.id, selectedSection?.page_start]);

  const refresh = React.useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: [...guidelineDocumentsQueryKey, versionId, "review"],
      }),
      queryClient.invalidateQueries({
        queryKey: [...guidelineDocumentsQueryKey, id],
      }),
    ]);
  }, [id, queryClient, versionId]);

  const action = useMutation({
    mutationFn: async (operation: () => Promise<unknown>) => operation(),
    onSuccess: async () => {
      await refresh();
    },
    onError: (error) =>
      showToast.error(
        "Review action failed",
        error instanceof Error ? error.message : "Unknown error",
      ),
  });

  async function saveSection() {
    if (!selectedSection) return;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.updateReviewSection(
        versionId,
        selectedSection.id,
        sectionDraft,
      ),
    );
    showToast.success("Section saved");
    setSectionDialogOpen(false);
  }

  async function moveSection(direction: -1 | 1) {
    if (!selectedSection) return;
    const index = sections.findIndex(
      (section) => section.id === selectedSection.id,
    );
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= sections.length) return;
    const reordered = [...sections];
    [reordered[index], reordered[nextIndex]] = [
      reordered[nextIndex],
      reordered[index],
    ];
    await action.mutateAsync(() =>
      GuidelineDocumentsService.reorderReviewSections(
        versionId,
        reordered.map((section, sortOrder) => ({
          id: section.id,
          parent_id: section.parent_id,
          level: section.level,
          sort_order: sortOrder,
        })),
      ),
    );
  }

  async function saveBlock() {
    if (!selectedBlock) return;
    let content: Record<string, unknown>;
    try {
      content = JSON.parse(blockJSON) as Record<string, unknown>;
    } catch {
      showToast.error(
        "Invalid JSON",
        "Correct the block payload before saving.",
      );
      return;
    }
    await action.mutateAsync(() =>
      GuidelineDocumentsService.updateReviewBlock(versionId, selectedBlock.id, {
        type: blockType,
        content,
      }),
    );
    showToast.success(
      "Block correction saved",
      "The review decision was reset because the clinical content changed.",
    );
  }

  function discardBlockChanges() {
    if (!selectedBlock) return;
    setBlockType(selectedBlock.type);
    setBlockJSON(JSON.stringify(selectedBlock.content, null, 2));
  }

  function changeBlockType(nextType: GuidelineBlockType) {
    setBlockType(nextType);
    let current: Record<string, unknown> = {};
    try {
      current = JSON.parse(blockJSON) as Record<string, unknown>;
    } catch {
      /* preserve an editable replacement */
    }
    const text = selectedBlock
      ? blockText({ ...selectedBlock, content: current })
      : "";
    let next: Record<string, unknown> = { ...current, type: nextType };
    if (["recommendation", "warning", "key_point"].includes(nextType)) {
      next = {
        type: nextType,
        title: String(current.title || nextType.replace("_", " ")),
        content: text,
        severity: String(current.severity || "standard"),
      };
    } else if (nextType === "paragraph" || nextType === "unknown") {
      next = { type: nextType, text };
    } else if (nextType === "heading") {
      next = { type: nextType, text, level: Number(current.level) || 2 };
    }
    setBlockJSON(JSON.stringify(next, null, 2));
  }

  async function decide(status: "reviewed" | "rejected") {
    if (!selectedBlock || action.isPending) return;
    if (blockDirty) {
      showToast.error(
        "Unsaved correction",
        "Save or discard your edits before recording a review decision.",
      );
      return;
    }
    const next = nextPendingAfter(selectedBlock.id);
    const remaining = pendingIndividualReviewBlocks.filter(
      (block) => block.id !== selectedBlock.id,
    ).length;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.reviewBlock(
        versionId,
        selectedBlock.id,
        status,
      ),
    );
    if (status === "reviewed") {
      showToast.success(
        selectedBlock.type === "figure" ? "Figure approved" : "Block approved",
        remaining === 0
          ? "All blocks requiring individual review are approved. Return to the Markdown editor and refresh the regeneration review."
          : `${remaining} block${remaining === 1 ? "" : "s"} still require individual review.`,
      );
      if (next) selectBlock(next);
      return;
    }
    showToast.error(
      "Block rejected",
      "A rejected block remains unavailable to readers and may block publication. Correct it, compare it with the source again, and approve the corrected block.",
    );
  }

  async function removeBlock() {
    if (
      !selectedBlock ||
      !window.confirm(
        "Delete this block? Use this only for extraction artifacts such as running headers or page numbers. This action is audited.",
      )
    )
      return;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.deleteReviewBlock(versionId, selectedBlock.id),
    );
    setSelectedBlockId(null);
  }

  async function splitSection() {
    if (!selectedSection || !selectedBlock) return;
    const title = window.prompt(
      "Title for the new section",
      `${selectedSection.title} — continued`,
    );
    if (!title?.trim()) return;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.splitReviewSection(
        versionId,
        selectedSection.id,
        {
          block_id: selectedBlock.id,
          title: title.trim(),
          level: selectedSection.level,
        },
      ),
    );
  }

  async function mergeSection() {
    if (!selectedSection) return;
    const index = sections.findIndex(
      (section) => section.id === selectedSection.id,
    );
    const target = sections[index - 1];
    if (
      !target ||
      !window.confirm(
        `Merge “${selectedSection.title}” into “${target.title}”?`,
      )
    )
      return;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.mergeReviewSection(
        versionId,
        selectedSection.id,
        target.id,
      ),
    );
    setSelectedSectionId(target.id);
    setSectionDialogOpen(false);
  }

  async function publish() {
    const validation =
      await GuidelineDocumentsService.validatePublication(versionId);
    await refresh();
    if (!validation.valid) {
      showToast.error(
        "Publication blocked",
        `${validation.errors.length} validation issue(s) require attention.`,
      );
      return;
    }
    if (
      !window.confirm(
        "Publish this reviewed version? The action is permanent and audited.",
      )
    )
      return;
    await action.mutateAsync(() =>
      GuidelineDocumentsService.publishVersion(versionId),
    );
    showToast.success("Guideline published");
    router.push(`/guidelines/${id}`);
  }

  const selectedIndex = selectedBlock
    ? visibleSectionBlocks.findIndex((block) => block.id === selectedBlock.id)
    : -1;
  const previousBlock =
    selectedIndex > 0 ? visibleSectionBlocks[selectedIndex - 1] : undefined;
  const nextBlock =
    visibleSectionBlocks[selectedIndex + 1] ||
    (blockReviewFilter === "all"
      ? undefined
      : nextPendingAfter(selectedBlock?.id));

  // Keyboard shortcuts read the latest handlers through a ref so the listener
  // is registered once.
  const shortcuts = React.useRef({
    next: () => {},
    previous: () => {},
    approve: () => {},
    reject: () => {},
  });
  React.useEffect(() => {
    shortcuts.current = {
      next: () => nextBlock && selectBlock(nextBlock),
      previous: () => previousBlock && selectBlock(previousBlock),
      approve: () => {
        if (selectedBlock && selectedBlock.review_status !== "reviewed")
          void decide("reviewed");
      },
      reject: () => {
        if (selectedBlock && selectedBlock.review_status !== "rejected")
          void decide("rejected");
      },
    };
  });
  React.useEffect(() => {
    if (tab !== "review" || sectionDialogOpen) return;
    function onKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (isEditableTarget(event.target)) return;
      const handler = {
        j: shortcuts.current.next,
        k: shortcuts.current.previous,
        a: shortcuts.current.approve,
        r: shortcuts.current.reject,
      }[event.key.toLowerCase()];
      if (!handler) return;
      event.preventDefault();
      handler();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [tab, sectionDialogOpen]);

  if (workspaceQuery.isLoading)
    return <LoadingState message="Loading editorial workspace..." />;
  if (workspaceQuery.isError || !workspace)
    return (
      <div className="p-6 text-destructive">
        {workspaceQuery.error instanceof Error
          ? workspaceQuery.error.message
          : "Review workspace unavailable."}
      </div>
    );

  const selectedPage =
    selectedBlock?.page_start || selectedSection?.page_start || pdfPage;
  const pdfSource = pdfUrl ? `${pdfUrl}#page=${selectedPage}` : null;
  const reviewedCount =
    individualReviewBlocks.length - pendingIndividualReviewBlocks.length;
  const reviewProgress = individualReviewBlocks.length
    ? Math.round((reviewedCount / individualReviewBlocks.length) * 100)
    : 100;
  const selectedSectionIndex = sections.findIndex(
    (section) => section.id === selectedSection?.id,
  );

  return (
    <div className="space-y-4">
      <div className="sticky top-16 z-30 space-y-3 rounded-lg border bg-background/95 p-3 shadow-sm backdrop-blur supports-[backdrop-filter]:bg-background/85">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex min-w-0 items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              aria-label="Back to guideline"
              onClick={() => router.push(`/guidelines/${id}`)}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div className="min-w-0">
              <h1 className="truncate text-lg font-semibold">
                Editorial review · Version {workspace.version.version}
              </h1>
              <p className="truncate text-xs text-muted-foreground">
                Compare each block with the source PDF, then approve or
                correct it.
              </p>
            </div>
            <Badge variant="outline" className="shrink-0 capitalize">
              {workspace.version.status.replaceAll("_", " ")}
            </Badge>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <GuidelineReviewIssuesPopover
              errors={workspace.validation.errors}
              warnings={workspace.validation.warnings}
              extractionWarnings={workspace.extraction_warnings}
              onFocusIssue={focusReviewIssue}
              onStartReview={startReview}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                action.mutate(() =>
                  GuidelineDocumentsService.validatePublication(versionId),
                )
              }
              disabled={action.isPending}
            >
              <Check className="h-4 w-4" /> Validate
            </Button>
            <Button
              size="sm"
              onClick={publish}
              disabled={
                action.isPending ||
                workspace.version.status === "published" ||
                !workspace.validation.valid
              }
            >
              <Send className="h-4 w-4" /> Publish
            </Button>
          </div>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <Progress
              value={reviewProgress}
              className="h-2 max-w-xs"
              aria-label="Individual review progress"
            />
            <span className="whitespace-nowrap text-sm text-muted-foreground">
              {individualReviewBlocks.length === 0
                ? "No blocks need individual review"
                : `${reviewedCount} of ${individualReviewBlocks.length} high-risk blocks approved`}
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            {pendingIndividualReviewBlocks.length > 0 ? (
              <Button size="sm" variant="secondary" onClick={startReview}>
                Review next pending
                <ChevronRight className="h-4 w-4" />
              </Button>
            ) : individualReviewBlocks.length > 0 ? (
              <span className="flex items-center gap-1 text-sm text-emerald-700 dark:text-emerald-400">
                <CircleCheck className="h-4 w-4" /> All high-risk blocks
                approved
              </span>
            ) : null}
            {cameFromRegeneration && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() =>
                  router.push(
                    `/guidelines/${id}/versions/${versionId}/markdown`,
                  )
                }
              >
                <ArrowLeft className="h-4 w-4" /> Regeneration review
              </Button>
            )}
          </div>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(value) => setTab(value as ReviewTab)}>
        <TabsList>
          <TabsTrigger value="review">Review</TabsTrigger>
          <TabsTrigger value="bulk">Bulk approve low-risk</TabsTrigger>
          <TabsTrigger value="completeness">Completeness report</TabsTrigger>
        </TabsList>

        <TabsContent value="review" className="mt-3 space-y-3">
          <div className="grid grid-cols-3 gap-1 rounded-md bg-muted p-1 xl:hidden">
            {(["sections", "source", "block"] as const).map((panel) => (
              <button
                key={panel}
                type="button"
                onClick={() => setCompactPanel(panel)}
                className={cn(
                  "rounded-sm px-2 py-1.5 text-sm capitalize",
                  compactPanel === panel && "bg-background font-medium shadow-sm",
                )}
              >
                {panel === "block" ? "Review block" : panel}
              </button>
            ))}
          </div>
          <div className="grid gap-4 xl:h-[calc(100vh-17rem)] xl:min-h-[560px] xl:grid-cols-[250px_minmax(0,1.1fr)_minmax(380px,1fr)]">
            <OutlinePanel
              className={cn(compactPanel !== "sections" && "hidden", "xl:flex")}
              sections={sections}
              selectedSectionId={selectedSection?.id}
              reviewCounts={sectionReviewCounts}
              onSelect={(sectionId) => {
                setSelectedSectionId(sectionId);
                setSelectedBlockId(null);
                setCompactPanel("block");
              }}
            />
            <SourcePanel
              className={cn(compactPanel !== "source" && "hidden", "xl:flex")}
              src={pdfSource}
              page={selectedPage}
              view={centerView}
              setView={setCenterView}
              previewBlocks={sectionBlocks.filter(
                (block) => block.review_status !== "rejected",
              )}
              assets={workspace.assets}
              mode={previewMode}
              setMode={setPreviewMode}
            />
            <BlockReviewPanel
              className={cn(compactPanel !== "block" && "hidden", "xl:flex")}
              section={selectedSection}
              blocks={visibleSectionBlocks}
              sectionBlockCount={sectionBlocks.length}
              assets={workspace.assets}
              blockReviewFilter={blockReviewFilter}
              setBlockReviewFilter={setBlockReviewFilter}
              selectedBlock={selectedBlock}
              selectedIndex={selectedIndex}
              onSelectBlock={setSelectedBlockId}
              onPrevious={previousBlock ? () => selectBlock(previousBlock) : undefined}
              onNext={nextBlock ? () => selectBlock(nextBlock) : undefined}
              onOpenSectionSettings={() => setSectionDialogOpen(true)}
              blockType={blockType}
              setBlockType={changeBlockType}
              blockJSON={blockJSON}
              setBlockJSON={setBlockJSON}
              dirty={blockDirty}
              pending={action.isPending}
              saveBlock={saveBlock}
              discardChanges={discardBlockChanges}
              decide={decide}
              removeBlock={removeBlock}
              splitSection={splitSection}
            />
          </div>
        </TabsContent>

        <TabsContent value="bulk" className="mt-3">
          <GuidelineBulkReviewPanel
            versionId={versionId}
            sections={sections}
            eligibleTypes={
              workspace.block_review_policy?.bulk_review_eligible_types || []
            }
            availableTypes={blockTypes}
            onSelectBlock={(block) => {
              setTab("review");
              setBlockReviewFilter("all");
              setCompactPanel("block");
              selectBlock(block);
            }}
            onReviewed={refresh}
          />
        </TabsContent>

        <TabsContent value="completeness" className="mt-3">
          <GuidelineCompletenessReport versionId={versionId} />
        </TabsContent>
      </Tabs>

      <Dialog open={sectionDialogOpen} onOpenChange={setSectionDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Section settings</DialogTitle>
            <DialogDescription>
              Fix the heading, level or position of a section that was
              extracted incorrectly.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-[1fr_110px]">
              <div className="space-y-1.5">
                <Label htmlFor="section-title">Section title</Label>
                <Input
                  id="section-title"
                  value={sectionDraft.title}
                  onChange={(event) =>
                    setSectionDraft((value) => ({
                      ...value,
                      title: event.target.value,
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="section-level">Heading level</Label>
                <Input
                  id="section-level"
                  type="number"
                  min={1}
                  max={6}
                  value={sectionDraft.level}
                  onChange={(event) =>
                    setSectionDraft((value) => ({
                      ...value,
                      level: Number(event.target.value),
                    }))
                  }
                />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="section-slug">Slug</Label>
              <Input
                id="section-slug"
                value={sectionDraft.slug}
                onChange={(event) =>
                  setSectionDraft((value) => ({
                    ...value,
                    slug: event.target.value,
                  }))
                }
              />
            </div>
            <div className="flex flex-wrap gap-2 border-t pt-3">
              <Button
                size="sm"
                variant="outline"
                onClick={() => moveSection(-1)}
                disabled={action.isPending || selectedSectionIndex <= 0}
              >
                <ArrowUp className="h-3.5 w-3.5" /> Move up
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => moveSection(1)}
                disabled={
                  action.isPending ||
                  selectedSectionIndex >= sections.length - 1
                }
              >
                <ArrowDown className="h-3.5 w-3.5" /> Move down
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={mergeSection}
                disabled={action.isPending || selectedSectionIndex <= 0}
              >
                <GitMerge className="h-3.5 w-3.5" /> Merge into previous
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSectionDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={saveSection} disabled={action.isPending}>
              <Save className="h-4 w-4" /> Save section
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function OutlinePanel({
  className,
  sections,
  selectedSectionId,
  reviewCounts,
  onSelect,
}: {
  className?: string;
  sections: GuidelineSectionRecord[];
  selectedSectionId?: string;
  reviewCounts: Map<string, { total: number; pending: number }>;
  onSelect: (sectionId: string) => void;
}) {
  const navRef = React.useRef<HTMLElement>(null);
  React.useEffect(() => {
    navRef.current
      ?.querySelector('[aria-current="true"]')
      ?.scrollIntoView({ block: "nearest" });
  }, [selectedSectionId]);
  return (
    <Card
      className={cn(
        "min-h-0 flex-col gap-0 overflow-hidden py-0 max-xl:h-[70vh]",
        className,
      )}
    >
      <div className="border-b px-3 py-2.5 text-sm font-medium">
        Sections{" "}
        <span className="font-normal text-muted-foreground">
          · {sections.length}
        </span>
      </div>
      <nav
        ref={navRef}
        aria-label="Guideline sections"
        className="min-h-0 flex-1 space-y-0.5 overflow-auto p-2"
      >
        {sections.map((section) => {
          const counts = reviewCounts.get(section.id);
          const selected = section.id === selectedSectionId;
          return (
            <button
              key={section.id}
              type="button"
              onClick={() => onSelect(section.id)}
              aria-current={selected ? "true" : undefined}
              title={`${section.title || "Untitled"} · page ${section.page_start || "—"}`}
              className={cn(
                "flex w-full items-center gap-2 rounded-md py-1.5 pr-2 text-left text-sm hover:bg-muted",
                selected && "bg-primary/10 font-medium text-primary hover:bg-primary/10",
              )}
              style={{
                paddingLeft: `${8 + Math.max(0, section.level - 1) * 12}px`,
              }}
            >
              <span className="min-w-0 flex-1 truncate">
                {section.title || "Untitled"}
              </span>
              {counts && counts.pending > 0 ? (
                <span
                  className="rounded-full bg-amber-100 px-1.5 text-xs font-medium text-amber-900 dark:bg-amber-900/40 dark:text-amber-200"
                  aria-label={`${counts.pending} pending review`}
                >
                  {counts.pending}
                </span>
              ) : counts ? (
                <CircleCheck
                  className="h-3.5 w-3.5 shrink-0 text-emerald-600"
                  aria-label="All reviewed"
                />
              ) : null}
            </button>
          );
        })}
      </nav>
    </Card>
  );
}

function SourcePanel({
  className,
  src,
  page,
  view,
  setView,
  previewBlocks,
  assets,
  mode,
  setMode,
}: {
  className?: string;
  src: string | null;
  page: number;
  view: "source" | "preview";
  setView: (view: "source" | "preview") => void;
  previewBlocks: GuidelineContentBlockRecord[];
  assets: GuidelineAssetRecord[];
  mode: "web" | "mobile";
  setMode: (mode: "web" | "mobile") => void;
}) {
  return (
    <Card
      className={cn(
        "min-h-0 flex-col gap-0 overflow-hidden py-0 max-xl:h-[75vh]",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex rounded-md bg-muted p-0.5 text-sm">
          <button
            type="button"
            onClick={() => setView("source")}
            className={cn(
              "flex items-center gap-1.5 rounded-sm px-2.5 py-1",
              view === "source" && "bg-background font-medium shadow-sm",
            )}
          >
            <FileText className="h-3.5 w-3.5" /> Source PDF · p.{page}
          </button>
          <button
            type="button"
            onClick={() => setView("preview")}
            className={cn(
              "rounded-sm px-2.5 py-1",
              view === "preview" && "bg-background font-medium shadow-sm",
            )}
          >
            Reader preview
          </button>
        </div>
        {view === "source" ? (
          src && (
            <Button asChild size="icon" variant="ghost" aria-label="Open PDF in a new tab">
              <a href={src} target="_blank" rel="noreferrer">
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          )
        ) : (
          <span className="flex gap-1">
            <Button
              size="icon"
              variant={mode === "web" ? "secondary" : "ghost"}
              aria-label="Web preview"
              onClick={() => setMode("web")}
            >
              <Monitor className="h-4 w-4" />
            </Button>
            <Button
              size="icon"
              variant={mode === "mobile" ? "secondary" : "ghost"}
              aria-label="Mobile preview"
              onClick={() => setMode("mobile")}
            >
              <Smartphone className="h-4 w-4" />
            </Button>
          </span>
        )}
      </div>
      {view === "source" ? (
        src ? (
          <iframe
            key={src}
            title="Original guideline PDF"
            src={src}
            className="min-h-0 w-full flex-1 border-0"
          />
        ) : (
          <div className="flex flex-1 items-center justify-center p-6 text-center text-sm text-muted-foreground">
            Original PDF preview could not be loaded.
          </div>
        )
      ) : (
        <div className="min-h-0 flex-1 overflow-auto bg-muted/30 p-3">
          <div
            className={cn(
              "mx-auto space-y-4 bg-background p-5 shadow-sm transition-all",
              mode === "mobile" ? "max-w-[390px] rounded-[24px]" : "w-full rounded-md",
            )}
          >
            {previewBlocks.length ? (
              previewBlocks.map((block) => (
                <BlockPreview
                  key={block.id}
                  block={block}
                  asset={findBlockAsset(block, assets)}
                />
              ))
            ) : (
              <div className="py-20 text-center text-sm text-muted-foreground">
                No active blocks to preview.
              </div>
            )}
          </div>
        </div>
      )}
    </Card>
  );
}

function findBlockAsset(
  block: GuidelineContentBlockRecord,
  assets: GuidelineAssetRecord[],
) {
  const assetId =
    typeof block.content.asset_id === "string" ? block.content.asset_id : "";
  return assets.find((item) => item.id === assetId);
}

function reviewStatusVariant(status: string) {
  if (status === "reviewed") return "default" as const;
  if (status === "rejected") return "destructive" as const;
  return "secondary" as const;
}

type BlockReviewPanelProps = {
  className?: string;
  section?: GuidelineSectionRecord;
  blocks: GuidelineContentBlockRecord[];
  sectionBlockCount: number;
  assets: GuidelineAssetRecord[];
  blockReviewFilter: BlockReviewFilter;
  setBlockReviewFilter: (value: BlockReviewFilter) => void;
  selectedBlock?: GuidelineContentBlockRecord;
  selectedIndex: number;
  onSelectBlock: (blockId: string) => void;
  onPrevious?: () => void;
  onNext?: () => void;
  onOpenSectionSettings: () => void;
  blockType: GuidelineBlockType;
  setBlockType: (value: GuidelineBlockType) => void;
  blockJSON: string;
  setBlockJSON: (value: string) => void;
  dirty: boolean;
  pending: boolean;
  saveBlock: () => void;
  discardChanges: () => void;
  decide: (status: "reviewed" | "rejected") => void;
  removeBlock: () => void;
  splitSection: () => void;
};

function BlockReviewPanel(props: BlockReviewPanelProps) {
  const block = props.selectedBlock;
  const [editorOpen, setEditorOpen] = React.useState(false);
  React.useEffect(() => {
    if (props.dirty) setEditorOpen(true);
  }, [props.dirty]);

  return (
    <Card
      className={cn(
        "min-h-0 flex-col gap-0 overflow-hidden py-0 max-xl:h-[80vh]",
        props.className,
      )}
    >
      <div className="space-y-2 border-b p-3">
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <div className="text-xs text-muted-foreground">Section</div>
            <div className="truncate font-medium" title={props.section?.title}>
              {props.section?.title || "No section selected"}
            </div>
          </div>
          {props.section && (
            <Button
              size="sm"
              variant="ghost"
              onClick={props.onOpenSectionSettings}
            >
              <Settings2 className="h-4 w-4" /> Section
            </Button>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Label className="sr-only" htmlFor="block-review-filter">
            Block review filter
          </Label>
          <select
            id="block-review-filter"
            aria-label="Block review filter"
            className="h-8 flex-1 rounded-md border bg-background px-2 text-xs"
            value={props.blockReviewFilter}
            onChange={(event) =>
              props.setBlockReviewFilter(
                event.target.value as BlockReviewFilter,
              )
            }
          >
            <option value="pending-high-risk">Pending review only</option>
            <option value="high-risk">All high-risk blocks</option>
            <option value="all">All blocks</option>
          </select>
          <span className="whitespace-nowrap text-xs text-muted-foreground">
            {props.blocks.length} of {props.sectionBlockCount}
          </span>
        </div>
      </div>

      {props.blocks.length > 0 && (
        <div className="max-h-36 shrink-0 overflow-auto border-b p-1.5">
          {props.blocks.map((item, index) => (
            <button
              key={item.id}
              type="button"
              onClick={() => props.onSelectBlock(item.id)}
              className={cn(
                "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-xs hover:bg-muted",
                block?.id === item.id && "bg-primary/10 text-primary hover:bg-primary/10",
              )}
            >
              <span className="w-5 shrink-0 text-right text-muted-foreground">
                {index + 1}
              </span>
              <span
                className={cn(
                  "h-2 w-2 shrink-0 rounded-full",
                  item.review_status === "reviewed"
                    ? "bg-emerald-500"
                    : item.review_status === "rejected"
                      ? "bg-destructive"
                      : "bg-amber-400",
                )}
                aria-label={item.review_status}
              />
              <span className="w-20 shrink-0 truncate capitalize text-muted-foreground">
                {item.type.replaceAll("_", " ")}
              </span>
              <span className="min-w-0 flex-1 truncate">
                {blockText(item) || "Structured payload"}
              </span>
            </button>
          ))}
        </div>
      )}

      <div className="min-h-0 flex-1 space-y-3 overflow-auto p-3">
        {!block ? (
          <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
            {props.sectionBlockCount === 0
              ? "This section has no blocks."
              : "No blocks in this section match the filter. Pick another section or show all blocks."}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-1.5">
                <Badge variant="outline" className="capitalize">
                  {block.type.replaceAll("_", " ")}
                </Badge>
                <Badge variant={reviewStatusVariant(block.review_status)}>
                  {block.review_status}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  p.{block.page_start || "—"} · confidence{" "}
                  {block.extraction_confidence == null
                    ? "—"
                    : `${Math.round(block.extraction_confidence * 100)}%`}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-xs text-muted-foreground">
                  {props.selectedIndex + 1}/{props.blocks.length}
                </span>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  aria-label="Previous block (K)"
                  disabled={!props.onPrevious}
                  onClick={props.onPrevious}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  aria-label="Next block (J)"
                  disabled={!props.onNext}
                  onClick={props.onNext}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {block.type === "figure" && (
              <div className="rounded-md border border-blue-200 bg-blue-50 p-3 text-xs leading-5 text-blue-950 dark:border-blue-900 dark:bg-blue-950/30 dark:text-blue-100">
                Compare the image, caption, alternative text, source,
                attribution and licence with the source document. Approving
                this figure also approves its linked asset.
              </div>
            )}

            <div className="rounded-md border bg-background p-3">
              <BlockPreview
                block={block}
                asset={findBlockAsset(block, props.assets)}
              />
            </div>

            <Collapsible open={editorOpen} onOpenChange={setEditorOpen}>
              <CollapsibleTrigger asChild>
                <Button size="sm" variant="ghost" className="group -ml-2">
                  <ChevronRight className="h-4 w-4 transition-transform group-data-[state=open]:rotate-90" />
                  <Pencil className="h-3.5 w-3.5" /> Correct extraction
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-2 space-y-3 rounded-md border p-3">
                <div className="space-y-1.5">
                  <Label htmlFor="block-content-type">Content type</Label>
                  <select
                    id="block-content-type"
                    className="h-9 w-full rounded-md border bg-background px-3 text-sm"
                    value={props.blockType}
                    onChange={(event) =>
                      props.setBlockType(
                        event.target.value as GuidelineBlockType,
                      )
                    }
                  >
                    {blockTypes.map((type) => (
                      <option key={type} value={type}>
                        {type.replaceAll("_", " ")}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="block-json">Typed JSON payload</Label>
                  <Textarea
                    id="block-json"
                    className="min-h-56 font-mono text-xs"
                    value={props.blockJSON}
                    onChange={(event) => props.setBlockJSON(event.target.value)}
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    onClick={props.saveBlock}
                    disabled={props.pending || !props.dirty}
                  >
                    <Save className="h-3.5 w-3.5" /> Save correction
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={props.discardChanges}
                    disabled={!props.dirty}
                  >
                    Discard
                  </Button>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </>
        )}
      </div>

      {block && (
        <div className="space-y-2 border-t bg-muted/30 p-3">
          {props.dirty && (
            <p className="text-xs text-amber-800 dark:text-amber-300">
              You have an unsaved correction. Save or discard it before
              approving.
            </p>
          )}
          <div className="flex items-center gap-2">
            <Button
              className="flex-1"
              onClick={() => props.decide("reviewed")}
              disabled={
                props.pending ||
                props.dirty ||
                block.review_status === "reviewed"
              }
            >
              <Check className="h-4 w-4" />
              {block.review_status === "reviewed"
                ? "Approved"
                : block.type === "figure"
                  ? "Approve figure & next"
                  : "Approve & next"}
              <kbd className="ml-1 hidden rounded border border-current/30 px-1 text-[10px] opacity-70 sm:inline">
                A
              </kbd>
            </Button>
            <Button
              variant="outline"
              onClick={() => props.decide("rejected")}
              disabled={
                props.pending ||
                props.dirty ||
                block.review_status === "rejected"
              }
            >
              <X className="h-4 w-4" /> Reject
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" aria-label="More block actions">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={props.splitSection}>
                  <Scissors className="h-4 w-4" /> Start new section here
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onSelect={props.removeBlock}
                >
                  <Trash2 className="h-4 w-4" /> Delete block
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <p className="hidden text-[11px] text-muted-foreground sm:block">
            Shortcuts: <kbd>J</kbd>/<kbd>K</kbd> next/previous ·{" "}
            <kbd>A</kbd> approve · <kbd>R</kbd> reject
          </p>
        </div>
      )}
    </Card>
  );
}
