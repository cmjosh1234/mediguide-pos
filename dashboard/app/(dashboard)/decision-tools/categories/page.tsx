"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { Pencil, Plus, Trash2 } from "lucide-react"
import { PageHeader } from "@/components/ui/page-header"
import { BackendDataTable } from "@/components/ui/backend-data-table"
import { usePermissionContext } from "@/lib/permission-context"
import { showToast } from "@/lib/toast"
import {
  TOOL_CATEGORIES_COLLECTION,
  toolCategoryService,
  type ToolCategory,
  type ToolCategoryInput,
} from "@/services/tool-category.service"
import type { FieldOption, RowAction } from "@/types/data-table"
import { toolCategoryColumns } from "./columns"
import { ToolCategoryDialog } from "./tool-category-dialog"

const availableFields: FieldOption[] = [
  {
    label: "Status",
    value: "status",
    type: "select",
    options: [
      { label: "Active", value: "active" },
      { label: "Inactive", value: "inactive" },
    ],
  },
]

export default function ToolCategoriesPage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { hasPermission, loading } = usePermissionContext()
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [editing, setEditing] = React.useState<ToolCategory | null>(null)

  React.useEffect(() => {
    if (loading) return
    if (!hasPermission("content", "read:any")) router.replace("/")
  }, [loading, hasPermission, router])

  // Refreshes the table and every category picker that shares this key.
  const refresh = React.useCallback(
    () => queryClient.invalidateQueries({ queryKey: ["backend", TOOL_CATEGORIES_COLLECTION] }),
    [queryClient],
  )

  const openDialog = React.useCallback((category: ToolCategory | null) => {
    setEditing(category)
    setDialogOpen(true)
  }, [])

  const save = async (input: ToolCategoryInput) => {
    try {
      if (editing) {
        await toolCategoryService.update(editing.id, input)
        showToast.success("Category updated", `"${input.name}" was saved`)
      } else {
        await toolCategoryService.create(input)
        showToast.success("Category added", `"${input.name}" was created`)
      }
      setDialogOpen(false)
      await refresh()
    } catch (error) {
      showToast.error("Could not save category", error instanceof Error ? error.message : "Try again")
    }
  }

  const rowActions = React.useMemo<RowAction<ToolCategory>[]>(
    () => [
      { id: "edit", label: "Edit", icon: Pencil, onClick: (category) => openDialog(category) },
      {
        id: "delete",
        label: "Delete",
        icon: Trash2,
        variant: "destructive",
        separator: true,
        // The backend also refuses, so tools are never left pointing at nothing.
        disabled: (category) => category.tool_count > 0,
        confirmMessage: "Delete this category? This cannot be undone.",
        onClick: async (category) => {
          await toolCategoryService.delete(category.id)
          showToast.success("Category deleted", `"${category.name}" was deleted`)
          await refresh()
        },
      },
    ],
    [openDialog, refresh],
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title="Tool Categories"
        description="Group calculators, decision tools and checklists by clinical area. A category with tools assigned cannot be deleted; mark it inactive instead."
        actions={
          hasPermission("content", "create:any")
            ? [{ label: "Add Category", onClick: () => openDialog(null), icon: <Plus className="h-4 w-4" /> }]
            : []
        }
      />
      <BackendDataTable<ToolCategory>
        collection={TOOL_CATEGORIES_COLLECTION}
        loadPage={toolCategoryService.listTable}
        columns={toolCategoryColumns}
        searchFields={["name", "slug", "description"]}
        rowActions={rowActions}
        availableFields={availableFields}
      />
      <ToolCategoryDialog open={dialogOpen} onOpenChange={setDialogOpen} category={editing} onSubmit={save} />
    </div>
  )
}
