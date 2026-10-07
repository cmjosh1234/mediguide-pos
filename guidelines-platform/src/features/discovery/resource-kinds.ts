import {
  BookOpen,
  Calculator,
  ChartLine,
  ExternalLink,
  FileText,
  type LucideIcon,
  Pill,
  Siren,
  Workflow,
} from "lucide-react";

// Each public content type gets its own label, icon, tint and call to action so
// a guideline, a live outbreak and a situation report read differently at a
// glance. `kind` selects the tint in discovery.css. Keep this table in step
// with the mobile app's resource_kind.dart.
export type ResourceKind = {
  label: string;
  Icon: LucideIcon;
  kind: string;
  action: string;
};

const KINDS: Record<string, ResourceKind> = {
  guideline: { label: "Guideline", Icon: BookOpen, kind: "guideline", action: "Read guideline" },
  outbreak: { label: "Outbreak", Icon: Siren, kind: "outbreak", action: "Open outbreak" },
  outbreak_document: { label: "Response document", Icon: FileText, kind: "outbreak", action: "Read document" },
  situation_report: { label: "Situation report", Icon: ChartLine, kind: "report", action: "Read report" },
  clinical_tool: { label: "Clinical tool", Icon: Calculator, kind: "tool", action: "Open tool" },
  algorithm: { label: "Algorithm", Icon: Workflow, kind: "tool", action: "Open algorithm" },
  drug_reference: { label: "Drug reference", Icon: Pill, kind: "drug", action: "Open drug reference" },
  approved_external_url: { label: "External resource", Icon: ExternalLink, kind: "neutral", action: "Visit website" },
};

export function resourceKind(contentType: string): ResourceKind {
  const known = KINDS[contentType];
  if (known) return known;
  const words = contentType.replaceAll("_", " ").trim();
  return {
    label: words ? words[0].toUpperCase() + words.slice(1) : "Resource",
    Icon: FileText,
    kind: "neutral",
    action: "Open",
  };
}
