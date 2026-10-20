import { AlertTriangle, ShieldAlert, ShieldCheck } from "lucide-react"
import { cn } from "@/lib/utils"

export type RiskLevel = "standard" | "high" | "critical"

export const riskLevelOptions: Array<{
  value: RiskLevel
  label: string
  description: string
  /** The review rule the backend enforces at this level. */
  rule: string
}> = [
  {
    value: "standard",
    label: "Standard",
    description: "Reference or routine scoring, such as BMI or a due date",
    rule: "The author may approve and publish their own versions.",
  },
  {
    value: "high",
    label: "High",
    description: "Informs treatment decisions, such as GCS or a fever pathway",
    rule: "A different person must approve each version.",
  },
  {
    value: "critical",
    label: "Critical",
    description: "An error could directly harm a patient, such as dosing or triage",
    rule: "A different person must approve and publish each version.",
  },
]

const riskRank: Record<RiskLevel, number> = { standard: 1, high: 2, critical: 3 }

export function normalizeRiskLevel(value: unknown): RiskLevel {
  return value === "high" || value === "critical" ? value : "standard"
}

export function isLowerRisk(next: RiskLevel, current: RiskLevel): boolean {
  return riskRank[next] < riskRank[current]
}

export function riskLevelOption(value: unknown) {
  const level = normalizeRiskLevel(value)
  return riskLevelOptions.find((option) => option.value === level)!
}

const badgeStyles: Record<RiskLevel, string> = {
  standard: "border-border text-muted-foreground",
  high: "border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200",
  critical: "border-red-300 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-200",
}

const badgeIcons = { standard: ShieldCheck, high: AlertTriangle, critical: ShieldAlert }

export function RiskLevelBadge({ level, className }: { level: unknown; className?: string }) {
  const option = riskLevelOption(level)
  const Icon = badgeIcons[option.value]
  return (
    <span
      title={option.rule}
      className={cn(
        "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium",
        badgeStyles[option.value],
        className,
      )}
    >
      <Icon className="h-3 w-3" />
      {option.label} risk
    </span>
  )
}

/** Banner shown while editing or reviewing a high or critical tool. */
export function RiskLevelNotice({ level }: { level: unknown }) {
  const option = riskLevelOption(level)
  if (option.value === "standard") return null
  return (
    <div role="note" className={cn("flex items-start gap-2 rounded-md border p-3 text-sm", badgeStyles[option.value])}>
      <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
      <p>
        <span className="font-semibold">{option.label} risk tool.</span> {option.rule}
      </p>
    </div>
  )
}
