"use client";

import * as React from "react";
import { AlertTriangle, ChevronDown, ChevronRight, Info, ShieldCheck } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { GuidelineReviewIssue } from "@/services/guideline-documents.service";

// Issues that only mean "a reviewer has not looked at this yet". They are the
// normal state of a draft, so they are summarised instead of listed as errors.
const reviewQueueIssueCodes: Record<string, (count: number) => string> = {
  unreviewed_high_risk_block: (count) =>
    `${count} high-risk block${count === 1 ? "" : "s"} awaiting publisher review`,
  unreviewed_clinical_asset: (count) =>
    `${count} clinical image${count === 1 ? "" : "s"} awaiting publisher review`,
};

export type ReviewIssueGroup = {
  code: string;
  title: string;
  remediation?: string;
  reviewQueue: boolean;
  issues: GuidelineReviewIssue[];
};

export function groupReviewIssues(
  issues: GuidelineReviewIssue[],
): ReviewIssueGroup[] {
  const groups = new Map<string, ReviewIssueGroup>();
  for (const issue of issues) {
    const existing = groups.get(issue.code);
    if (existing) {
      existing.issues.push(issue);
      continue;
    }
    groups.set(issue.code, {
      code: issue.code,
      title: issue.message,
      remediation: issue.remediation,
      reviewQueue: issue.code in reviewQueueIssueCodes,
      issues: [issue],
    });
  }
  for (const group of groups.values()) {
    const summarise = reviewQueueIssueCodes[group.code];
    if (summarise) group.title = summarise(group.issues.length);
    else if (group.issues.length > 1)
      group.title = `${group.issues[0].message} (+${group.issues.length - 1} more)`;
  }
  // Real defects first; the review queue is progress, not a failure.
  return [...groups.values()].sort(
    (left, right) => Number(left.reviewQueue) - Number(right.reviewQueue),
  );
}

export function GuidelineReviewIssuesPopover({
  errors,
  warnings,
  extractionWarnings,
  onFocusIssue,
  onStartReview,
}: {
  errors: GuidelineReviewIssue[];
  warnings: GuidelineReviewIssue[];
  extractionWarnings: string[];
  onFocusIssue: (issue: GuidelineReviewIssue) => void;
  onStartReview: () => void;
}) {
  const [open, setOpen] = React.useState(false);
  const groups = React.useMemo(() => groupReviewIssues(errors), [errors]);
  const advisoryCount = warnings.length + extractionWarnings.length;
  const defectCount = groups
    .filter((group) => !group.reviewQueue)
    .reduce((total, group) => total + group.issues.length, 0);
  const hasBlockers = errors.length > 0;

  if (!hasBlockers && advisoryCount === 0)
    return (
      <Badge className="gap-1">
        <ShieldCheck className="h-3.5 w-3.5" /> Ready to publish
      </Badge>
    );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className={
            defectCount > 0
              ? "border-destructive/50 text-destructive"
              : hasBlockers
                ? "border-amber-400 text-amber-800 dark:text-amber-300"
                : ""
          }
        >
          <AlertTriangle className="h-4 w-4" />
          {hasBlockers
            ? `${groups.length} blocker${groups.length === 1 ? "" : "s"}`
            : `${advisoryCount} advisor${advisoryCount === 1 ? "y" : "ies"}`}
          <ChevronDown className="h-3.5 w-3.5" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[min(92vw,460px)] p-0">
        <div className="border-b px-4 py-3">
          <div className="text-sm font-semibold">Publication checklist</div>
          <p className="text-xs text-muted-foreground">
            Everything below must be resolved before this version can be
            published.
          </p>
        </div>
        <div className="max-h-[60vh] space-y-2 overflow-auto p-3">
          {groups.map((group) => (
            <IssueGroup
              key={group.code}
              group={group}
              onFocusIssue={(issue) => {
                setOpen(false);
                onFocusIssue(issue);
              }}
              onStartReview={() => {
                setOpen(false);
                onStartReview();
              }}
            />
          ))}
          {advisoryCount > 0 && (
            <Collapsible className="rounded-md border">
              <CollapsibleTrigger className="group flex w-full items-center gap-2 px-3 py-2 text-left text-sm">
                <Info className="h-4 w-4 shrink-0 text-muted-foreground" />
                <span className="flex-1">
                  {advisoryCount} advisor{advisoryCount === 1 ? "y" : "ies"}{" "}
                  <span className="text-muted-foreground">
                    (do not block publication)
                  </span>
                </span>
                <ChevronRight className="h-4 w-4 transition-transform group-data-[state=open]:rotate-90" />
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1 border-t px-3 py-2 text-xs text-muted-foreground">
                {extractionWarnings.map((warning, index) => (
                  <p key={`extraction-${index}`}>{warning}</p>
                ))}
                {warnings.map((issue, index) => (
                  <p key={`${issue.code}-${index}`}>{issue.message}</p>
                ))}
              </CollapsibleContent>
            </Collapsible>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function IssueGroup({
  group,
  onFocusIssue,
  onStartReview,
}: {
  group: ReviewIssueGroup;
  onFocusIssue: (issue: GuidelineReviewIssue) => void;
  onStartReview: () => void;
}) {
  const tone = group.reviewQueue
    ? "border-amber-300 bg-amber-50/60 dark:bg-amber-950/20"
    : "border-destructive/40 bg-destructive/5";
  return (
    <Collapsible className={`rounded-md border ${tone}`}>
      <div className="flex items-start gap-2 px-3 py-2">
        <AlertTriangle
          className={`mt-0.5 h-4 w-4 shrink-0 ${group.reviewQueue ? "text-amber-600" : "text-destructive"}`}
        />
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium">{group.title}</div>
          {group.remediation && (
            <p className="mt-0.5 text-xs text-muted-foreground">
              {group.remediation}
            </p>
          )}
          <div className="mt-2 flex flex-wrap gap-2">
            {group.reviewQueue && (
              <Button size="sm" className="h-7" onClick={onStartReview}>
                Start reviewing
              </Button>
            )}
            {group.issues.length > 1 ? (
              <CollapsibleTrigger asChild>
                <Button size="sm" variant="ghost" className="group h-7 px-2">
                  <ChevronRight className="h-3.5 w-3.5 transition-transform group-data-[state=open]:rotate-90" />
                  Show {group.issues.length}
                </Button>
              </CollapsibleTrigger>
            ) : (
              <Button
                size="sm"
                variant="ghost"
                className="h-7 px-2"
                onClick={() => onFocusIssue(group.issues[0])}
              >
                Go to issue
              </Button>
            )}
          </div>
        </div>
      </div>
      {group.issues.length > 1 && (
        <CollapsibleContent className="max-h-64 overflow-auto border-t">
          {group.issues.map((issue, index) => (
            <button
              key={`${issue.block_id || issue.asset_id || issue.section_id || ""}-${index}`}
              type="button"
              className="block w-full truncate px-3 py-1.5 text-left text-xs hover:bg-muted"
              title={issue.message}
              onClick={() => onFocusIssue(issue)}
            >
              {issue.message}
            </button>
          ))}
        </CollapsibleContent>
      )}
    </Collapsible>
  );
}
