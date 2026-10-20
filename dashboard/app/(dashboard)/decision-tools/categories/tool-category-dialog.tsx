"use client"

import * as React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { IconPicker } from "@/components/ui/icon-picker"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { isValidStoredIcon } from "@/components/ui/stored-icon"
import { Textarea } from "@/components/ui/textarea"
import type { ToolCategory, ToolCategoryInput } from "@/services/tool-category.service"

const schema = z.object({
  name: z.string().trim().min(1, "Name is required").max(100),
  slug: z
    .string()
    .trim()
    .regex(/^([a-z0-9]+(-[a-z0-9]+)*)?$/, "Use lowercase letters, numbers and hyphens"),
  description: z.string().max(500),
  icon: z.string().refine(isValidStoredIcon, "Choose an icon from the list or paste an emoji"),
  color: z.string().regex(/^(#[0-9a-fA-F]{6})?$/, "Use a hex colour such as #0B63CE"),
  sort_order: z.number().int().min(0),
  status: z.enum(["active", "inactive"]),
})

type FormValues = z.infer<typeof schema>

interface ToolCategoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The category being edited; omit to create one. */
  category?: ToolCategory | null
  onSubmit: (input: ToolCategoryInput) => Promise<void>
}

function defaults(category?: ToolCategory | null): FormValues {
  return {
    name: category?.name ?? "",
    slug: category?.slug ?? "",
    description: category?.description ?? "",
    icon: category?.icon ?? "",
    color: category?.color ?? "",
    sort_order: category?.sort_order ?? 0,
    status: category?.status ?? "active",
  }
}

export function ToolCategoryDialog({ open, onOpenChange, category, onSubmit }: ToolCategoryDialogProps) {
  const editing = Boolean(category)
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: defaults(category) })

  React.useEffect(() => {
    if (open) reset(defaults(category))
  }, [open, category, reset])

  const icon = watch("icon")
  const color = watch("color")
  const status = watch("status")

  const submit = async (values: FormValues) => {
    await onSubmit({
      ...values,
      // An empty slug is generated from the name by the backend.
      slug: values.slug || undefined,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? "Edit tool category" : "Add tool category"}</DialogTitle>
          <DialogDescription>
            Categories group calculators, decision tools and checklists by clinical area.
          </DialogDescription>
        </DialogHeader>
        <form id="tool-category-form" onSubmit={handleSubmit(submit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="category-name">Name *</Label>
            <Input id="category-name" {...register("name")} placeholder="Maternal & Newborn" />
            {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="category-slug">Slug</Label>
            <Input id="category-slug" {...register("slug")} placeholder="Generated from the name" />
            {errors.slug && <p className="text-sm text-destructive">{errors.slug.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="category-description">Description</Label>
            <Textarea id="category-description" rows={3} {...register("description")} />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="category-icon">Icon</Label>
              <IconPicker
                id="category-icon"
                value={icon}
                onChange={(value) => setValue("icon", value, { shouldDirty: true, shouldValidate: true })}
                invalid={Boolean(errors.icon)}
              />
              {errors.icon && <p className="text-sm text-destructive">{errors.icon.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="category-color">Colour</Label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  aria-label="Pick a colour"
                  value={color || "#0B63CE"}
                  onChange={(event) => setValue("color", event.target.value.toUpperCase(), { shouldDirty: true, shouldValidate: true })}
                  className="h-9 w-10 cursor-pointer rounded border bg-transparent"
                />
                <Input id="category-color" {...register("color")} placeholder="#0B63CE" />
              </div>
              {errors.color && <p className="text-sm text-destructive">{errors.color.message}</p>}
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="category-order">Display order</Label>
              <Input id="category-order" type="number" min={0} {...register("sort_order", { valueAsNumber: true })} />
              {errors.sort_order && <p className="text-sm text-destructive">Use a whole number of 0 or more</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="category-status">Status</Label>
              <Select value={status} onValueChange={(value) => setValue("status", value as FormValues["status"], { shouldDirty: true })}>
                <SelectTrigger id="category-status" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="inactive">Inactive (hidden when choosing a category)</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="tool-category-form" disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {editing ? "Save changes" : "Add category"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
