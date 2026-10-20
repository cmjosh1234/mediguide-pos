import { Badge } from "@/components/ui/badge"
import type { ClinicalToolDefinition } from "@/services/clinical-tool.service"

type OutlineRow = { title: string; items: string[]; empty: string }

function outlineRows(definition: ClinicalToolDefinition): OutlineRow[] {
  return [
    { title: "Inputs", items: definition.inputs.map((input) => input.label || input.key), empty: "No inputs" },
    { title: "Checklist sections", items: definition.sections.map((section) => section.title || section.key), empty: "No sections" },
    { title: "Calculations", items: definition.calculation.map((step) => step.key), empty: "No calculations" },
    { title: "Rules", items: definition.rules.map((rule) => rule.key), empty: "No rules" },
    { title: "Results", items: definition.outputs.map((output) => output.label || output.key), empty: "No results" },
    { title: "Interpretations", items: definition.interpretations.map((item) => item.label || item.key), empty: "No interpretations" },
    { title: "Warnings", items: (definition.warnings ?? []).map((warning) => warning.text || warning.key), empty: "No warnings" },
    { title: "Citations", items: (definition.citations ?? []).map((citation) => citation.title || citation.key), empty: "No citations" },
    { title: "Test cases", items: definition.test_cases.map((testCase) => testCase.key), empty: "No test cases: add some before submitting" },
  ]
}

/** Read-only summary of a definition; editing happens in the JSON editor. */
export function DefinitionOutline({ definition }: { definition?: ClinicalToolDefinition }) {
  if (!definition) {
    return (
      <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
        The outline appears once the editor contains a valid definition.
      </p>
    )
  }
  const details = [
    ["Title", definition.title],
    ["Type", definition.tool_type.replaceAll("_", " ")],
    ["Version", definition.version],
    ["Language", definition.locale],
    ["Clinical owner", definition.clinical_owner],
    ["Clinical reviewer", definition.clinical_reviewer],
  ].filter((entry): entry is [string, string] => Boolean(entry[1]))

  return (
    <div className="space-y-5">
      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-3">
        {details.map(([label, value]) => (
          <div key={label}>
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="text-sm font-medium">{value}</dd>
          </div>
        ))}
      </dl>
      <ul className="divide-y rounded-md border">
        {outlineRows(definition).map((row) => (
          <li key={row.title} className="flex flex-col gap-2 p-3 sm:flex-row sm:items-start">
            <span className="flex w-44 shrink-0 items-center gap-2 text-sm font-medium">
              {row.title}
              <Badge variant="secondary" className="px-1.5 py-0 text-xs">{row.items.length}</Badge>
            </span>
            {row.items.length > 0 ? (
              <span className="flex flex-wrap gap-1.5">
                {row.items.map((item, index) => (
                  <span key={`${item}-${index}`} className="rounded bg-muted px-2 py-0.5 text-xs">
                    {item}
                  </span>
                ))}
              </span>
            ) : (
              <span className="text-xs text-muted-foreground">{row.empty}</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
