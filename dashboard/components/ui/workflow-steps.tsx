import { Check, Circle } from "lucide-react"
import { cn } from "@/lib/utils"

/**
 * Progress through a review workflow (e.g. Draft → In review → Approved →
 * Published). Earlier steps are ticked and the current one is highlighted, so
 * every publication workflow in the dashboard reads the same way.
 */
export function WorkflowSteps({
  steps,
  current,
  className,
}: {
  steps: readonly string[]
  /** Index of the current step. */
  current: number
  className?: string
}) {
  return (
    <ol aria-label="Workflow progress" className={cn("flex flex-wrap items-center gap-1 text-sm", className)}>
      {steps.map((step, index) => {
        const done = index < current
        const active = index === current
        return (
          <li key={step} className="flex items-center gap-1">
            {index > 0 && <span aria-hidden className={cn("h-px w-6", done || active ? "bg-primary" : "bg-border")} />}
            <span
              aria-current={active ? "step" : undefined}
              className={cn(
                "flex items-center gap-1.5 rounded-full px-2.5 py-1",
                active && "bg-primary text-primary-foreground",
                done && "text-foreground",
                !active && !done && "text-muted-foreground",
              )}
            >
              {done ? <Check className="h-3.5 w-3.5" /> : <Circle className={cn("h-2 w-2", active && "fill-current")} />}
              {step}
            </span>
          </li>
        )
      })}
    </ol>
  )
}
