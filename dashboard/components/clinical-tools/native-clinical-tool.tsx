"use client";

import * as React from "react";
import {
  previewClinicalTool,
  clinicalToolInitialInputs,
  clinicalToolInputVisible,
  type ClinicalToolPreviewResult,
} from "@/lib/clinical-tool-evaluator";
import type { ClinicalToolDefinition } from "@/services/clinical-tool.service";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";

function inputValue(value: unknown): unknown {
  if (typeof value === "object" && value !== null && "value" in value) {
    return (value as { value: unknown }).value;
  }
  return value;
}

export function NativeClinicalTool({
  definition,
}: {
  definition: ClinicalToolDefinition;
}) {
  const [inputs, setInputs] = React.useState<Record<string, unknown>>(() => clinicalToolInitialInputs(definition));
  const [units, setUnits] = React.useState<Record<string, string>>(() =>
    Object.fromEntries(
      definition.inputs.map((field) => [
        field.key,
        field.default_unit ?? field.allowed_units?.[0] ?? "",
      ]),
    ),
  );
  const [result, setResult] = React.useState<ClinicalToolPreviewResult | null>(
    null,
  );
  const [error, setError] = React.useState<string | null>(null);
  React.useEffect(() => {
    setInputs(clinicalToolInitialInputs(definition));
    setUnits(Object.fromEntries(definition.inputs.map((field) => [field.key, field.default_unit ?? field.allowed_units?.[0] ?? ""])));
    setResult(null);
    setError(null);
  }, [definition]);
  const feedbackRef = React.useRef<HTMLDivElement>(null);
  React.useEffect(() => {
    if (result || error) feedbackRef.current?.scrollIntoView?.({ behavior: "smooth", block: "start" });
  }, [result, error]);
  const setValue = (key: string, value: unknown) => {
    setInputs((current) => ({ ...current, [key]: value }));
    setResult(null);
    setError(null);
  };
  const run = (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const submitted = Object.fromEntries(Object.entries(inputs).filter(([, value]) => value !== ""));
      setResult(previewClinicalTool(definition, submitted));
      setError(null);
    } catch (cause) {
      setResult(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "The tool could not be evaluated.",
      );
    }
  };

  const visibleFields = definition.inputs.filter((field) => clinicalToolInputVisible(definition, field.key, inputs));
  const groups = [
    ...[...definition.sections].sort((a, b) => a.order - b.order).map((section) => ({ ...section, fields: visibleFields.filter((field) => field.section_key === section.key) })),
    { key: "ungrouped", title: "", fields: visibleFields.filter((field) => !definition.sections.some((section) => section.key === field.section_key)) },
  ].filter((group) => group.fields.length > 0);
  const renderInput = (field: ClinicalToolDefinition["inputs"][number]) => {
          const id = `clinical-tool-${field.key}`;
          if (field.control === "textarea")
            return (
              <div key={field.key} className="space-y-2">
                <Label className="block leading-relaxed" htmlFor={id}>{field.label}{field.required ? " *" : ""}</Label>
                <Textarea className="min-h-28 text-base" id={id} rows={3} required={field.required}
                  value={String(inputs[field.key] ?? "")}
                  onChange={(event) => setValue(field.key, event.target.value)} />
              </div>
            );
          if (field.control === "radio" && field.options?.length)
            return (
              <fieldset key={field.key} className="space-y-2">
                <legend className="text-sm font-semibold leading-relaxed">{field.label}{field.required ? " *" : ""}</legend>
                {field.options.map((option, index) => (
                  <label key={index} htmlFor={`${id}-${index}`} className={`flex min-h-12 cursor-pointer items-start gap-3 rounded-xl border p-3 leading-relaxed transition-colors ${inputs[field.key] === option.value ? "border-primary bg-primary/5" : "border-input hover:bg-muted/50"}`}>
                    <input id={`${id}-${index}`} name={id} className="mt-1 size-5 shrink-0 accent-primary" type="radio" required={field.required}
                      checked={inputs[field.key] === option.value}
                      onChange={() => setValue(field.key, option.value)} />
                    {option.label}
                  </label>
                ))}
              </fieldset>
            );
          if (field.type === "boolean" || field.type === "checklist_item")
            return (
              <label
                key={field.key}
                htmlFor={id}
                className={`flex min-h-14 cursor-pointer items-start gap-3 rounded-xl border p-4 leading-relaxed transition-colors ${inputs[field.key] === true ? "border-primary bg-primary/5" : "border-input hover:bg-muted/50"}`}
              >
                <input
                  id={id}
                  type="checkbox"
                  className="mt-0.5 size-5 shrink-0 accent-primary"
                  checked={inputs[field.key] === true}
                  onChange={(event) =>
                    setValue(field.key, event.target.checked)
                  }
                />
                <span>
                  <span className="font-medium">{field.label}</span>
                  {field.required ? (
                    <span aria-label="required"> *</span>
                  ) : null}
                </span>
              </label>
            );
          if (field.options?.length)
            return (
              <div key={field.key} className="space-y-2">
                <Label className="block leading-relaxed" htmlFor={id}>
                  {field.label}
                  {field.required ? " *" : ""}
                </Label>
                <select
                  id={id}
                  required={field.required}
                  className="border-input bg-background min-h-12 w-full min-w-0 rounded-xl border px-3 py-3 text-base focus-visible:outline-2 focus-visible:outline-ring"
                  value={String(inputs[field.key] ?? "")}
                  onChange={(event) => {
                    const option = field.options?.find(
                      (item) => String(item.value) === event.target.value,
                    );
                    setValue(field.key, option?.value ?? event.target.value);
                  }}
                >
                  <option value="">Select an option</option>
                  {field.options.map((option) => (
                    <option
                      key={String(option.value)}
                      value={String(option.value)}
                    >
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            );
          const numeric = ["number", "integer", "measurement"].includes(
            field.type,
          );
          return (
            <div key={field.key} className="space-y-2">
              <Label className="block leading-relaxed" htmlFor={id}>
                {field.label}
                {field.required ? " *" : ""}
              </Label>
              <div className="flex min-w-0 flex-wrap items-start gap-3">
                <Input
                  id={id}
                  className="h-12 min-w-0 flex-1 basis-40 rounded-xl text-base md:text-base"
                  inputMode={numeric ? field.type === "integer" ? "numeric" : "decimal" : undefined}
                  required={field.required}
                  type={
                    field.type === "date" ? "date" : field.type === "time" ? "time" : numeric ? "number" : "text"
                  }
                  min={field.allowed_units?.length && units[field.key] !== field.default_unit ? undefined : field.minimum}
                  max={field.allowed_units?.length && units[field.key] !== field.default_unit ? undefined : field.maximum}
                  step={field.step ?? (field.type === "integer" ? 1 : "any")}
                  value={String(inputValue(inputs[field.key]) ?? "")}
                  onChange={(event) =>
                    setValue(
                      field.key,
                      numeric && event.target.value !== ""
                        ? field.allowed_units?.length
                          ? {
                              value: Number(event.target.value),
                              unit: units[field.key],
                            }
                          : Number(event.target.value)
                        : event.target.value,
                    )
                  }
                />
                {field.allowed_units?.length ? (
                  <select
                    aria-label={`${field.label} unit`}
                    className="border-input bg-background min-h-12 max-w-full min-w-28 rounded-xl border px-3 py-3 text-base"
                    value={units[field.key]}
                    onChange={(event) => {
                      const unit = event.target.value;
                      setUnits((current) => ({
                        ...current,
                        [field.key]: unit,
                      }));
                      const value = inputValue(inputs[field.key]);
                      if (value !== undefined && value !== "") {
                        setValue(field.key, { value, unit });
                      }
                    }}
                  >
                    {field.allowed_units.map((unit) => (
                      <option key={unit} value={unit}>
                        {unit}
                      </option>
                    ))}
                  </select>
                ) : field.default_unit ? (
                  <span className="flex min-h-12 shrink-0 items-center rounded-xl border px-3 text-base text-muted-foreground">
                    {field.default_unit}
                  </span>
                ) : null}
              </div>
            </div>
          );
  };

  return (
    <div className="min-w-0 space-y-5" data-testid="native-clinical-tool">
      {definition.warnings
        ?.filter((warning) => !warning.when)
        .map((warning) => (
          <Alert
            key={warning.key}
            variant={
              warning.severity === "critical" ? "destructive" : "default"
            }
          >
            <AlertTitle>Clinical warning</AlertTitle>
            <AlertDescription>{warning.text}</AlertDescription>
          </Alert>
        ))}
      <p className="text-sm text-muted-foreground">Required fields are marked *</p>
      <form className="min-w-0 space-y-5" onSubmit={run} noValidate>
        {groups.map((group) => (
          <fieldset key={group.key} className="min-w-0 space-y-5 rounded-2xl border bg-background p-4 shadow-sm sm:p-5">
            {group.title ? <legend className="max-w-full px-2 text-base font-semibold leading-relaxed">{group.title}</legend> : null}
            {group.fields.map((field) => (
              <div key={field.key} className="min-w-0 space-y-2">
                {renderInput(field)}
                {field.help_text ? <p className="text-sm text-muted-foreground">{field.help_text}</p> : null}
                {field.clinical_warning ? <p className="text-sm font-semibold leading-relaxed">{field.clinical_warning}</p> : null}
              </div>
            ))}
          </fieldset>
        ))}
        {error ? (
          <div ref={feedbackRef} className="scroll-mt-24"><Alert variant="destructive">
            <AlertTitle>Unable to calculate</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert></div>
        ) : null}
        <div className="sticky bottom-0 z-10 grid grid-cols-1 gap-2 rounded-xl border bg-background p-3 shadow-sm sm:grid-cols-[1fr_auto]" style={{ paddingBottom: "max(0.75rem, env(safe-area-inset-bottom))" }}>
          <Button className="h-auto min-h-12 whitespace-normal py-3 text-base" type="submit">
            {definition.tool_type === "checklist"
              ? "Review checklist"
              : "Calculate"}
          </Button>
          <Button
            type="button"
            variant="outline"
            className="h-auto min-h-12 py-3 text-base"
            onClick={() => {
              if (definition.completion.reset_confirmation && !window.confirm("Reset all responses?")) return;
              setInputs(clinicalToolInitialInputs(definition));
              setUnits(
                Object.fromEntries(
                  definition.inputs.map((field) => [
                    field.key,
                    field.default_unit ?? field.allowed_units?.[0] ?? "",
                  ]),
                ),
              );
              setResult(null);
              setError(null);
            }}
          >
            Reset
          </Button>
        </div>
      </form>
      {result ? (
        <section
          ref={result ? feedbackRef : undefined}
          className="min-w-0 space-y-4 rounded-2xl border bg-muted/30 p-4 sm:p-5"
          aria-live="polite"
        >
          <h3 className="text-lg font-semibold">Results</h3>
          {Object.entries(result.values).map(([key, value]) => {
            const output = definition.outputs.find((item) => item.key === key);
            return (
              <div key={key} className="space-y-1 rounded-xl border bg-background p-3 break-words">
                <span className="block text-sm font-medium text-muted-foreground">{output?.label ?? key}: </span>
                <span className="block text-xl font-semibold tabular-nums">
                  {typeof value === "number" && output?.precision !== undefined ? value.toFixed(output.precision) : String(value ?? "—")}
                  {output?.unit ? ` ${output.unit}` : ""}
                </span>
              </div>
            );
          })}
          {result.interpretation ? (
            <p>
              <span className="font-medium">Interpretation: </span>
              {result.interpretation}
            </p>
          ) : null}
          {result.recommendations.length ? (
            <div>
              <h4 className="font-medium">Recommendations</h4>
              <ul className="list-disc pl-5">
                {result.recommendations.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {result.warnings.length ? (
            <Alert variant="destructive">
              <AlertTitle>Warnings</AlertTitle>
              <AlertDescription>
                <ul className="list-disc pl-5">
                  {result.warnings.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          ) : null}
          {result.checklist ? (
            <p>
              {result.checklist.completed_required} of{" "}
              {result.checklist.total_required} required items complete (
              {result.checklist.percentage}%).
            </p>
          ) : null}
        </section>
      ) : null}
      {definition.citations?.length ? (
        <section className="space-y-2">
          <h3 className="font-semibold">Clinical references</h3>
          <ul className="list-disc pl-5 text-sm">
            {definition.citations.map((citation) => (
              <li key={citation.key}>
                {citation.url ? (
                  <a
                    className="underline"
                    href={citation.url}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {citation.title}
                  </a>
                ) : (
                  citation.title
                )}
                {citation.organization ? ` — ${citation.organization}` : ""}
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
