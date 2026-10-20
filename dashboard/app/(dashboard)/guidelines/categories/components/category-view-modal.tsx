"use client"

import * as React from "react"
import { format } from "date-fns"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { GuidelineCategoriesWithParent } from "@/types/expanded"
import { StoredIcon } from "@/components/ui/stored-icon"

interface CategoryViewModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  category: GuidelineCategoriesWithParent | null
  onEdit?: (category: GuidelineCategoriesWithParent) => void
}

function Row({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="grid grid-cols-3 gap-3 py-2 border-b last:border-b-0">
      <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide pt-0.5">
        {label}
      </div>
      <div className="col-span-2 text-sm break-words">{children}</div>
    </div>
  )
}

function formatDate(value?: string) {
  if (!value) return "—"
  try {
    return format(new Date(value), "PPpp")
  } catch {
    return value
  }
}

export function CategoryViewModal({
  open,
  onOpenChange,
  category,
  onEdit,
}: CategoryViewModalProps) {
  if (!category) return null

  const parent = category.expand?.parent_category
  const emptyDash = <span className="text-muted-foreground">—</span>

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {category.name}
            <Badge
              variant={category.status === "active" ? "default" : "secondary"}
              className="text-xs"
            >
              {category.status}
            </Badge>
          </DialogTitle>
          <DialogDescription>Category details</DialogDescription>
        </DialogHeader>

        <div className="rounded-md border bg-muted/20 px-4 py-2">
          <Row label="Slug">
            {category.slug ? (
              <code className="text-xs bg-background px-1.5 py-0.5 rounded border">
                /{category.slug.replace(/^\//, "")}
              </code>
            ) : (
              emptyDash
            )}
          </Row>
          <Row label="Description">
            {category.description ? (
              <p className="whitespace-pre-wrap text-sm">{category.description}</p>
            ) : (
              emptyDash
            )}
          </Row>
          <Row label="Parent">
            {parent ? (
              parent.name
            ) : category.parent_category ? (
              <code className="text-xs">{category.parent_category}</code>
            ) : (
              <span className="text-muted-foreground">Root Category</span>
            )}
          </Row>
          <Row label="Order">
            {typeof category.sort_order === "number" ? category.sort_order : emptyDash}
          </Row>
          <Row label="Color">
            {category.color ? (
              <span className="inline-flex items-center gap-2">
                <span
                  className="inline-block w-4 h-4 rounded-full border"
                  style={{ backgroundColor: category.color }}
                  aria-hidden
                />
                <code className="text-xs">{category.color}</code>
              </span>
            ) : (
              emptyDash
            )}
          </Row>
          <Row label="Icon">
            {category.icon ? (
              <span className="flex items-center gap-2">
                <StoredIcon icon={category.icon} fallback={null} />
                <code className="text-xs">{category.icon}</code>
              </span>
            ) : (
              emptyDash
            )}
          </Row>
          <Row label="Created">{formatDate(category.created)}</Row>
          <Row label="Updated">{formatDate(category.updated)}</Row>
          <Row label="ID">
            <code className="text-xs">{category.id}</code>
          </Row>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          {onEdit && (
            <Button
              onClick={() => {
                onOpenChange(false)
                onEdit(category)
              }}
            >
              Edit Category
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
