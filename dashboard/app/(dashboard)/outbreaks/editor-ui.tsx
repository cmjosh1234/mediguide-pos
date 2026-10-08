"use client";

// Pieces shared by the outbreak and situation report editors, so both pages
// present their publication workflow and form errors the same way.

import * as React from "react";
import { AlertCircle, CheckCircle2, Save } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { BackendRequestError } from "@/lib/backend-client";
import { cn } from "@/lib/utils";
import { RequiredMark } from "./required-field";

// The fields a backend validation error names (outbreak_validation.go), so
// they can be highlighted.
export function errorFields(value: unknown): Array<{ field: string; message: string }> {
  if (!(value instanceof BackendRequestError)) return [];
  const fields = (value.meta as { fields?: unknown } | undefined)?.fields;
  return Array.isArray(fields)
    ? fields.filter(
        (item): item is { field: string; message: string } =>
          typeof item?.field === "string" && typeof item?.message === "string",
      )
    : [];
}

export const WORKFLOW_STEPS = ["Draft", "In review", "Approved", "Published"];
// A correction is applied to the published record when approved, then removed.
export const CORRECTION_STEPS = ["Draft", "In review", "Applied"];

// -1 means withdrawn (outside the forward path).
export function workflowStage(status?: string, approvedAt?: string) {
  if (!status || status === "draft") return 0;
  if (status === "pending_review") return approvedAt ? 2 : 1;
  if (status === "withdrawn") return -1;
  return 3;
}

export function WorkflowSteps({ stage, steps = WORKFLOW_STEPS }: { stage: number; steps?: string[] }) {
  if (stage < 0) return <Badge variant="destructive">Withdrawn</Badge>;
  return (
    <ol className="flex items-center gap-2 text-sm">
      {steps.map((step, index) => {
        const done = index < stage || stage === steps.length - 1;
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
            {index < steps.length - 1 ? (
              <span className="h-px w-6 bg-border" aria-hidden="true" />
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}

export function UnsavedChanges({
  message,
  saving,
  onSave,
  className,
}: {
  message: string;
  saving: boolean;
  onSave: () => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex w-full flex-wrap items-center gap-3 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200",
        className,
      )}
    >
      <AlertCircle className="h-4 w-4 shrink-0" />
      <span className="min-w-0 flex-1">{message}</span>
      <Button size="sm" disabled={saving} onClick={onSave}>
        <Save className="mr-2 h-4 w-4" />
        Save draft
      </Button>
    </div>
  );
}

export function Field({
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
export function SelectField({
  id,
  label,
  value,
  onChange,
  options,
  empty,
  error,
  disabled,
}: {
  id?: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ id: string; name: string }>;
  /** Label of a blank first option; omit when a value is always chosen. */
  empty?: string;
  error?: string;
  disabled?: boolean;
}) {
  return (
    <div>
      <Label htmlFor={id}>{label}</Label>
      <select
        id={id}
        className="mt-2 h-10 w-full rounded-md border bg-background px-3 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40"
        value={value}
        disabled={disabled}
        aria-invalid={Boolean(error)}
        aria-describedby={error && id ? `${id}-error` : undefined}
        onChange={(event) => onChange(event.target.value)}
      >
        {empty === undefined ? null : <option value="">{empty}</option>}
        {options.map((option) => (
          <option key={option.id} value={option.id}>
            {option.name}
          </option>
        ))}
      </select>
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
