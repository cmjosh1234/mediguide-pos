"use client"

import * as React from "react"
import { Calculator } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/datatable-column-header"
import { StoredIcon } from "@/components/ui/stored-icon"
import type { ExtendedColumnDef } from "@/types/data-table"
import type { ToolCategory } from "@/services/tool-category.service"

export const toolCategoryColumns: ExtendedColumnDef<ToolCategory>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Category" canSort={false} />,
    cell: ({ row }) => {
      const category = row.original
      return (
        <div className="flex items-center gap-2 font-medium">
          <span
            className="flex h-7 w-7 items-center justify-center rounded-md"
            style={category.color ? { backgroundColor: `${category.color}1A`, color: category.color } : undefined}
          >
            <StoredIcon icon={category.icon} fallback={<Calculator className="h-4 w-4" />} />
          </span>
          <span>{category.name}</span>
        </div>
      )
    },
  },
  {
    accessorKey: "tool_count",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Tools" canSort={false} />,
    cell: ({ row }) => <span className="text-sm">{row.original.tool_count}</span>,
  },
  {
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" canSort={false} />,
    cell: ({ row }) => (
      <Badge variant={row.original.status === "active" ? "default" : "secondary"}>
        {row.original.status === "active" ? "Active" : "Inactive"}
      </Badge>
    ),
  },
  {
    accessorKey: "description",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Description" canSort={false} />,
    cell: ({ row }) =>
      row.original.description ? (
        <div className="max-w-md truncate text-sm" title={row.original.description}>
          {row.original.description}
        </div>
      ) : (
        <span className="text-muted-foreground">—</span>
      ),
  },
  {
    accessorKey: "slug",
    defaultVisible: false,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Slug" canSort={false} />,
    cell: ({ row }) => <code className="text-xs">{row.original.slug}</code>,
  },
  {
    accessorKey: "sort_order",
    defaultVisible: false,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Order" canSort={false} />,
    cell: ({ row }) => <span className="text-sm">{row.original.sort_order}</span>,
  },
]
