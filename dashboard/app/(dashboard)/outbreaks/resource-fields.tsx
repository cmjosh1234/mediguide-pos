"use client";

import * as React from "react";
import Link from "next/link";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { DocumentKind } from "@/services/document-kinds.service";
import {
  outbreaksService,
  type ChildContentInput,
  type OutbreakResourceRecord,
  type PublishedGuidelineRecord,
  type SituationReportRecord,
} from "@/services/outbreaks.service";
import { RequiredMark } from "./required-field";

export const SITUATION_REPORT_TYPE = {
  label: "Situation report",
  description: "A published situation report already in the system.",
};

// Kinds that match an earlier built-in resource type keep that type's description.
export const KIND_DESCRIPTION_OVERRIDES: Record<string, string> = {
  guideline: "A published guideline already in the system.",
};

// Situation reports aren't document kinds; the prefix can't collide with a kind slug.
export const SITUATION_REPORT_OPTION = "@situation_report";

export type ResourceDraft = {
  title: string;
  description: string;
  issuing_organization: string;
  document_type: string;
  url: string;
};

export type ResourceProblem = { id: string; label: string; message: string };

export const emptyResourceDraft = (): ResourceDraft => ({
  title: "",
  description: "",
  issuing_organization: "",
  document_type: "",
  url: "",
});

/** Older resources (internal routes, external URLs) have no document type and must pick one when edited. */
export function draftFromResource(resource: OutbreakResourceRecord): ResourceDraft {
  const documentType =
    resource.resource_type === "situation_report"
      ? SITUATION_REPORT_OPTION
      : resource.resource_type === "guideline"
        ? resource.document_kind || "guideline"
        : "";
  return {
    title: resource.title || "",
    description: resource.description || "",
    issuing_organization: resource.issuing_organization || "",
    document_type: documentType,
    url: documentType ? resource.url || "" : "",
  };
}

export function resourcePayload(draft: ResourceDraft): ChildContentInput {
  const isReport = draft.document_type === SITUATION_REPORT_OPTION;
  return {
    title: draft.title,
    description: draft.description,
    issuing_organization: draft.issuing_organization,
    resource_type: isReport ? "situation_report" : "guideline",
    document_kind: isReport ? "other" : draft.document_type,
    url: draft.url,
  };
}

export function findResourceProblems(draft: ResourceDraft, idPrefix = "resource"): ResourceProblem[] {
  const problems: ResourceProblem[] = [];
  if (!draft.title.trim()) {
    problems.push({ id: `${idPrefix}-title`, label: "Title", message: "Title is required." });
  }
  if (!draft.document_type) {
    problems.push({ id: `${idPrefix}-type`, label: "Document type", message: "Select a document type." });
  } else if (!draft.url) {
    problems.push(
      draft.document_type === SITUATION_REPORT_OPTION
        ? { id: `${idPrefix}-url`, label: "Situation report", message: "Select a published situation report." }
        : { id: `${idPrefix}-url`, label: "Document", message: "Select a published document." },
    );
  }
  return problems;
}

export function documentTypeDescription(slug: string | undefined, kinds: DocumentKind[]) {
  if (!slug) return "";
  if (slug === SITUATION_REPORT_OPTION) return SITUATION_REPORT_TYPE.description;
  return KIND_DESCRIPTION_OVERRIDES[slug] ?? kinds.find((kind) => kind.slug === slug)?.description ?? "";
}

const selectClassName =
  "h-10 w-full rounded-md border bg-background px-3 aria-invalid:border-destructive aria-invalid:ring-[3px] aria-invalid:ring-destructive/20";

export function ResourceFields({
  draft,
  onChange,
  documentKinds,
  reports,
  errorFor,
  idPrefix = "resource",
  className = "grid items-end gap-2 md:grid-cols-2",
  allowSituationReports = true,
}: {
  draft: ResourceDraft;
  onChange: (update: (current: ResourceDraft) => ResourceDraft) => void;
  documentKinds: DocumentKind[];
  reports: SituationReportRecord[];
  errorFor: (id: string) => string | undefined;
  idPrefix?: string;
  className?: string;
  /** Offer published situation reports as a document type, as outbreaks do. */
  allowSituationReports?: boolean;
}) {
  const isReport = draft.document_type === SITUATION_REPORT_OPTION;
  const kindSlug = isReport ? "" : draft.document_type;
  const [kindDocuments, setKindDocuments] = React.useState<{
    slug: string;
    items: PublishedGuidelineRecord[];
  } | null>(null);

  React.useEffect(() => {
    if (!kindSlug) return;
    let cancelled = false;
    void outbreaksService
      .listPublishedGuidelines("", kindSlug)
      .then((page) => {
        if (!cancelled) setKindDocuments({ slug: kindSlug, items: page.items || [] });
      })
      .catch(() => {
        if (!cancelled) setKindDocuments({ slug: kindSlug, items: [] });
      });
    return () => {
      cancelled = true;
    };
  }, [kindSlug]);

  const selectedKind = documentKinds.find((kind) => kind.slug === kindSlug);
  const selectableKinds = documentKinds.filter(
    (kind) => kind.status === "active" || kind.slug === kindSlug,
  );
  const loading = Boolean(kindSlug) && kindDocuments?.slug !== kindSlug;
  const options = isReport
    ? reports
        .filter((report) => report.status === "published")
        .map((report) => ({ value: `/situation-reports/${report.id}`, label: report.title || "Untitled report" }))
    : !loading && kindDocuments
      ? kindDocuments.items.map((document) => ({
          value: `/public/guidelines/${document.id}`,
          label: document.title || "Untitled document",
        }))
      : [];
  if (draft.url && !loading && !options.some((option) => option.value === draft.url)) {
    options.unshift({ value: draft.url, label: "Current selection (no longer listed)" });
  }
  const description = documentTypeDescription(draft.document_type, documentKinds);
  const set = (key: keyof ResourceDraft) => (event: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const value = event.target.value;
    onChange((current) => ({ ...current, [key]: value }));
  };
  const id = (name: string) => `${idPrefix}-${name}`;

  return (
    <div className={className}>
      <Field id={id("title")} label="Title" required error={errorFor(id("title"))}>
        <Input
          id={id("title")}
          placeholder="Resource title"
          aria-required
          aria-invalid={Boolean(errorFor(id("title")))}
          value={draft.title}
          onChange={set("title")}
        />
      </Field>
      <Field id={id("type")} label="Document type" required error={errorFor(id("type"))}>
        <select
          id={id("type")}
          aria-required
          aria-invalid={Boolean(errorFor(id("type")))}
          className={selectClassName}
          value={draft.document_type}
          onChange={(event) => {
            const value = event.target.value;
            onChange((current) => ({ ...current, document_type: value, url: "" }));
          }}
        >
          <option value="">Select a document type</option>
          {selectableKinds.map((kind) => (
            <option key={kind.slug} value={kind.slug}>
              {kind.name}
            </option>
          ))}
          {allowSituationReports ? (
            <option value={SITUATION_REPORT_OPTION}>{SITUATION_REPORT_TYPE.label}</option>
          ) : null}
        </select>
        {description ? <p className="text-xs text-muted-foreground">{description}</p> : null}
      </Field>
      {draft.document_type ? (
        <Field
          id={id("url")}
          label={isReport ? "Published situation report" : `Published ${selectedKind?.name ?? "document"}`}
          required
          error={errorFor(id("url"))}
        >
          {options.length > 0 ? (
            <select
              id={id("url")}
              aria-required
              aria-invalid={Boolean(errorFor(id("url")))}
              className={selectClassName}
              value={draft.url}
              onChange={set("url")}
            >
              <option value="">
                {isReport ? "Select a published situation report" : "Select a published document"}
              </option>
              {options.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          ) : (
            <p className="flex min-h-10 items-center text-sm text-muted-foreground">
              {loading ? (
                "Loading documents…"
              ) : isReport ? (
                "No published situation reports are linked to this outbreak yet."
              ) : (
                <span>
                  No published {selectedKind?.name ?? "documents"} yet.{" "}
                  <Link href="/guidelines/create" className="underline">
                    Add one under Guidelines
                  </Link>
                  .
                </span>
              )}
            </p>
          )}
        </Field>
      ) : null}
      <Field id={id("issuer")} label="Issuing organization">
        <Input
          id={id("issuer")}
          placeholder="Issuing organization"
          value={draft.issuing_organization}
          onChange={set("issuing_organization")}
        />
      </Field>
      <Field id={id("description")} label="Description">
        <Input
          id={id("description")}
          placeholder="Short public description"
          value={draft.description}
          onChange={set("description")}
        />
      </Field>
    </div>
  );
}

export function Field({
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
