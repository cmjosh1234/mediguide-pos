"use client"

import * as React from "react"
import { FolderOpen, LucideIcon } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
    icon?: React.ReactNode
  }
  className?: string
}

export function EmptyState({
  icon: Icon = FolderOpen,
  title,
  description,
  action,
  className
}: EmptyStateProps) {
  return (
    <div role="status" aria-live="polite" className={cn(
      "flex flex-col items-center justify-center py-12 px-4 text-center",
      className
    )}>
      {Icon && (
        <Icon aria-hidden="true" className="h-12 w-12 text-muted-foreground/60 mb-4" />
      )}
      <h3 className="text-lg font-semibold text-foreground mb-2">
        {title}
      </h3>
      {description && (
        <p className="text-sm text-muted-foreground mb-6 max-w-sm">
          {description}
        </p>
      )}
      {action && (
        <Button onClick={action.onClick}>
          {action.icon && <span className="mr-2">{action.icon}</span>}
          {action.label}
        </Button>
      )}
    </div>
  )
}
