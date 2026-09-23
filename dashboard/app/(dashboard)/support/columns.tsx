"use client"

import { ColumnDef } from "@tanstack/react-table"
import { format } from "date-fns"
import Link from "next/link"

import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { DataTableColumnHeader } from "@/components/ui/datatable-column-header"

import type { SupportTicketsWithExpanded } from "@/types/expanded"
import { TICKET_STATUS_CONFIG, TICKET_PRIORITY_CONFIG } from "@/types/support-tickets"

export const columns: ColumnDef<SupportTicketsWithExpanded>[] = [
  {
    accessorKey: "subject",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Subject" />
    ),
    cell: ({ row }) => {
      const subject = row.getValue("subject") as string
      const ticket = row.original
      return (
        <div className="max-w-[300px]">
          <Link 
            href={`/support/${ticket.id}`}
            className="font-medium text-primary hover:underline truncate block"
          >
            {subject}
          </Link>
        </div>
      )
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const status = row.getValue("status") as keyof typeof TICKET_STATUS_CONFIG
      const statusConfig = TICKET_STATUS_CONFIG[status]
      
      return (
        <Badge variant="outline" className={statusConfig.color}>
          {statusConfig.label}
        </Badge>
      )
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },
  {
    accessorKey: "priority",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Priority" />
    ),
    cell: ({ row }) => {
      const priority = row.getValue("priority") as keyof typeof TICKET_PRIORITY_CONFIG
      const priorityConfig = TICKET_PRIORITY_CONFIG[priority]
      
      return (
        <Badge variant="outline" className={priorityConfig.color}>
          {priorityConfig.label}
        </Badge>
      )
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },
  {
    accessorKey: "category",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Category" />
    ),
    cell: ({ row }) => {
      const category = row.getValue("category") as string | undefined
      return category ? (
        <Badge variant="secondary">{category}</Badge>
      ) : (
        <span className="text-muted-foreground">-</span>
      )
    },
  },
  {
    accessorKey: "user_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created By" />
    ),
    cell: ({ row }) => {
      const ticket = row.original
      const user = ticket.expand?.user_id
      const isGuest = !ticket.user_id
      
      if (!user || (!user.name && !user.email)) {
        return <span className="text-muted-foreground">Unknown</span>
      }
      
      return (
        <div className="flex items-center gap-2">
          <Avatar className="h-6 w-6">
            <AvatarImage src={user.avatar} />
            <AvatarFallback>
              {user.name?.charAt(0) || user.email?.charAt(0) || "?"}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium truncate">
              {user.name || user.email}
            </p>
            {isGuest && (
              <p className="text-xs text-muted-foreground truncate">
                Guest · {user.email}
              </p>
            )}
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "assigned_to",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Assigned To" />
    ),
    cell: ({ row }) => {
      const ticket = row.original
      const assignedUser = ticket.expand?.assigned_to
      
      if (!assignedUser) {
        return <span className="text-muted-foreground">Unassigned</span>
      }
      
      return (
        <div className="flex items-center gap-2">
          <Avatar className="h-6 w-6">
            <AvatarImage src={assignedUser.avatar} />
            <AvatarFallback>
              {assignedUser.name?.charAt(0) || assignedUser.email.charAt(0)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium truncate">
              {assignedUser.name || assignedUser.email}
            </p>
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "created",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created" />
    ),
    cell: ({ row }) => {
      const date = row.getValue("created") as string
      return (
        <div className="text-sm text-muted-foreground">
          {format(new Date(date), "MMM d, yyyy")}
        </div>
      )
    },
  },
  {
    accessorKey: "updated",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Last Updated" />
    ),
    cell: ({ row }) => {
      const date = row.getValue("updated") as string
      return (
        <div className="text-sm text-muted-foreground">
          {format(new Date(date), "MMM d, yyyy")}
        </div>
      )
    },
  },
]