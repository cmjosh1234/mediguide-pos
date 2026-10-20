"use client"

import * as React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { CalculatorsResponse, CalculatorsTypeOptions, CalculatorsStatusOptions } from "@/types/backend-types"
import { AlertTriangle, Calculator, Brain, CheckSquare, ChevronRight, FileCode2 } from "lucide-react"
import { IconPicker } from "@/components/ui/icon-picker"
import { isValidStoredIcon, StoredIcon } from "@/components/ui/stored-icon"
import {
  isLowerRisk,
  normalizeRiskLevel,
  riskLevelOption,
  riskLevelOptions,
  type RiskLevel,
} from "@/components/clinical-tools/risk-level"
import { useToolCategories } from "@/hooks/use-tool-categories"
import { hasBackendPermission } from "@/lib/backend-client"
import { cn } from "@/lib/utils"

const decisionToolSchema = z.object({
  name: z.string().min(1, "Name is required").max(200, "Name must be less than 200 characters"),
  description: z.string().optional(),
  version: z.string().min(1, "Version is required").max(20, "Version must be less than 20 characters"),
  type: z.enum(["calculator", "decision_tool", "checklist"]),
  status: z.enum(["active", "draft", "archived"]),
  category_id: z.string().min(1, "Choose a category"),
  risk_level: z.enum(["standard", "high", "critical"]),
  appFile: z.string().optional(),
  icon: z
    .string()
    .optional()
    .refine((value) => !value || isValidStoredIcon(value), "Choose an icon from the list or paste an emoji"),
  color: z.string().optional(),
  backgroundColor: z.string().optional(),
})

type DecisionToolFormData = z.infer<typeof decisionToolSchema>

interface DecisionToolFormProps {
  initialData?: Partial<CalculatorsResponse>
  onSubmit: (data: DecisionToolFormData) => Promise<void>
  onCancel: () => void
  loading?: boolean
  mode: "create" | "edit"
}

const typeOptions = [
  { 
    value: "calculator", 
    label: "Calculator",
    description: "Mathematical calculators for clinical scoring",
    // Icon colour the mobile app uses when the tool sets none.
    appColor: "#075FBD",
    icon: <Calculator className="h-4 w-4" />
  },
  { 
    value: "decision_tool", 
    label: "Decision Tool",
    description: "Decision trees and clinical pathways",
    // Icon colour the mobile app uses when the tool sets none.
    appColor: "#168A59",
    icon: <Brain className="h-4 w-4" />
  },
  { 
    value: "checklist", 
    label: "Checklist",
    description: "Step-by-step clinical checklists",
    // Icon colour the mobile app uses when the tool sets none.
    appColor: "#246A9E",
    icon: <CheckSquare className="h-4 w-4" />
  },
]

const statusOptions = [
  { value: "draft", label: "Draft", description: "Under development, hidden from users", dot: "bg-amber-500" },
  { value: "active", label: "Active", description: "Listed in the mobile app", dot: "bg-emerald-500" },
  { value: "archived", label: "Archived", description: "No longer available for new use", dot: "bg-muted-foreground" },
]

const HEX_COLOR = /^#[0-9a-fA-F]{6}$/

function FieldError({ message }: { message?: string }) {
  return message ? <p className="text-sm text-destructive">{message}</p> : null
}

function ColorField({
  id,
  label,
  value,
  fallback,
  disabled,
  onChange,
  inputProps,
}: {
  id: string
  label: string
  value: string
  /** The colour the app falls back to while this one is unset. */
  fallback: string
  disabled: boolean
  onChange: (value: string) => void
  inputProps: React.ComponentProps<typeof Input>
}) {
  const isSet = HEX_COLOR.test(value)
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex gap-2">
        <input
          type="color"
          aria-label={`Pick ${label.toLowerCase()}`}
          title={isSet ? value : "Not set: using the default"}
          value={isSet ? value : fallback}
          onChange={(event) => onChange(event.target.value)}
          className={cn(
            "h-9 w-10 shrink-0 cursor-pointer rounded-md border border-input bg-transparent",
            !isSet && "border-dashed opacity-50",
          )}
          disabled={disabled}
        />
        <Input id={id} placeholder="Default" className="font-mono" disabled={disabled} {...inputProps} />
      </div>
    </div>
  )
}

export function DecisionToolForm({ 
  initialData, 
  onSubmit, 
  onCancel, 
  loading = false,
  mode 
}: DecisionToolFormProps) {
  const appFile = typeof initialData?.appFile === "string" ? initialData.appFile : ""

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setValue,
    watch,
  } = useForm<DecisionToolFormData>({
    resolver: zodResolver(decisionToolSchema),
    defaultValues: {
      name: initialData?.name || "",
      description: initialData?.description || "",
      version: initialData?.version || "1.0.0",
      type: initialData?.type || CalculatorsTypeOptions.calculator,
      status: initialData?.status || CalculatorsStatusOptions.draft,
      category_id: initialData?.category_id || "",
      risk_level: normalizeRiskLevel(initialData?.risk_level),
      appFile,
      icon: initialData?.icon || "",
      color: initialData?.color || "",
      backgroundColor: initialData?.backgroundColor || "",
    },
  })

  const watchedType = watch("type")
  const watchedStatus = watch("status")
  const watchedCategory = watch("category_id")
  const watchedRisk = watch("risk_level")
  const { activeCategories, byId: categoriesById, loading: categoriesLoading } = useToolCategories()
  // A retired category stays selectable on tools that already use it.
  const currentCategory = categoriesById.get(watchedCategory)
  const categoryChoices =
    currentCategory && currentCategory.status !== "active"
      ? [...activeCategories, currentCategory]
      : activeCategories
  // Only reviewers may lower an existing tool's risk; the backend enforces it.
  const savedRisk = normalizeRiskLevel(initialData?.risk_level)
  const canLowerRisk = mode === "create" || hasBackendPermission("calculator.review")
  const riskLocked = (level: RiskLevel) => !canLowerRisk && isLowerRisk(level, savedRisk)
  const watchedIcon = watch("icon")
  // Duplicates start without versions, so only an edited tool can have one.
  const hasPublishedVersion =
    mode === "edit" &&
    Boolean((initialData as { current_version_id?: string | null } | undefined)?.current_version_id)

  const onFormSubmit = async (data: DecisionToolFormData) => {
    try {
      // Include the app file from state
      const submitData = {
        ...data,
        appFile,
      }
      await onSubmit(submitData)
    } catch (error) {
      console.error("Form submission error:", error)
    }
  }

  const getTypeIcon = (type: string) => {
    const option = typeOptions.find(opt => opt.value === type)
    return option?.icon || <Calculator className="h-4 w-4" />
  }

  const isFormLoading = loading || isSubmitting

  const watchedName = watch("name")
  const watchedColor = watch("color") ?? ""
  const watchedBackground = watch("backgroundColor") ?? ""
  const typeOption = typeOptions.find((option) => option.value === watchedType)
  const statusOption = statusOptions.find((option) => option.value === watchedStatus)
  const riskOption = riskLevelOption(watchedRisk)
  // Mirrors the mobile tool tile: the icon colour falls back to the type's
  // colour, and the tile is a light tint of the tile or icon colour.
  const previewColor = HEX_COLOR.test(watchedColor) ? watchedColor : (typeOption?.appColor ?? "#075FBD")
  const previewBackground = HEX_COLOR.test(watchedBackground) ? `${watchedBackground}1F` : `${previewColor}1A`

  return (
    <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
              <CardDescription>What users see in tool lists and search.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="space-y-2">
                <Label htmlFor="name">Tool name *</Label>
                <Input
                  id="name"
                  {...register("name")}
                  placeholder="e.g. Glasgow Coma Scale"
                  className={errors.name ? "border-destructive" : ""}
                  disabled={isFormLoading}
                />
                <FieldError message={errors.name?.message} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  {...register("description")}
                  placeholder="Describe what this tool does and when to use it"
                  rows={3}
                  disabled={isFormLoading}
                />
              </div>
              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="type">Tool type *</Label>
                  <Select
                    value={watchedType}
                    onValueChange={(value) => setValue("type", value as CalculatorsTypeOptions)}
                    disabled={isFormLoading}
                  >
                    <SelectTrigger id="type" className="w-full">
                      <SelectValue>
                        <span className="flex items-center gap-2">
                          {getTypeIcon(watchedType)}
                          {typeOption?.label}
                        </span>
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {typeOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          <div className="flex items-start gap-3 py-1">
                            <span className="mt-0.5">{option.icon}</span>
                            <div>
                              <div className="font-medium">{option.label}</div>
                              <div className="text-xs text-muted-foreground">{option.description}</div>
                            </div>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="category_id">Category *</Label>
                  <Select
                    value={watchedCategory || undefined}
                    onValueChange={(value) => setValue("category_id", value, { shouldDirty: true, shouldValidate: true })}
                    disabled={isFormLoading || categoriesLoading}
                  >
                    <SelectTrigger id="category_id" className={cn("w-full", errors.category_id && "border-destructive")}>
                      <SelectValue placeholder={categoriesLoading ? "Loading categories…" : "Select a category"} />
                    </SelectTrigger>
                    <SelectContent>
                      {categoryChoices.map((category) => (
                        <SelectItem key={category.id} value={category.id}>
                          <span className="flex items-center gap-2">
                            <span style={category.color ? { color: category.color } : undefined}>
                              <StoredIcon icon={category.icon} fallback={<Calculator className="h-4 w-4" />} />
                            </span>
                            {category.name}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FieldError message={errors.category_id?.message} />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Review and availability</CardTitle>
              <CardDescription>How carefully changes are reviewed, and whether users can find the tool.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="risk_level">Risk level *</Label>
                  <Select
                    value={watchedRisk}
                    onValueChange={(value) => setValue("risk_level", value as RiskLevel, { shouldDirty: true })}
                    disabled={isFormLoading}
                  >
                    <SelectTrigger id="risk_level" className="w-full">
                      <SelectValue>{riskOption.label}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {riskLevelOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value} disabled={riskLocked(option.value)}>
                          <div className="py-1">
                            <div className="font-medium">{option.label}</div>
                            <div className="text-xs text-muted-foreground">{option.description}</div>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="status">Status *</Label>
                  <Select
                    value={watchedStatus}
                    onValueChange={(value) => setValue("status", value as CalculatorsStatusOptions)}
                    disabled={isFormLoading}
                  >
                    <SelectTrigger id="status" className="w-full">
                      <SelectValue>
                        <span className="flex items-center gap-2">
                          <span className={cn("h-2 w-2 rounded-full", statusOption?.dot)} />
                          {statusOption?.label}
                        </span>
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          <div className="flex items-start gap-2 py-1">
                            <span className={cn("mt-1.5 h-2 w-2 shrink-0 rounded-full", option.dot)} />
                            <div>
                              <div className="font-medium">{option.label}</div>
                              <div className="text-xs text-muted-foreground">{option.description}</div>
                            </div>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
                <span className="font-medium text-foreground">{riskOption.label} risk:</span> {riskOption.rule}
                {!canLowerRisk && savedRisk !== "standard" && (
                  <span className="mt-1 block">Only clinical reviewers can lower a tool&apos;s risk level.</span>
                )}
              </div>
              {watchedStatus === CalculatorsStatusOptions.active && !hasPublishedVersion && (
                <div role="alert" className="flex gap-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                  <p>
                    This tool has no published version yet. Active tools are listed in the mobile app,
                    but users can&apos;t open this one until a reviewed version is published from its
                    authoring page. Keep it as Draft until then.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6 lg:sticky lg:top-6 lg:self-start">
          <Card>
            <CardHeader>
              <CardTitle>Appearance</CardTitle>
              <CardDescription>How the tool looks in the mobile app.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div aria-label="Mobile preview" className="flex items-center gap-3 rounded-xl border bg-background p-3">
                <span
                  className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl"
                  style={{ color: previewColor, backgroundColor: previewBackground }}
                >
                  <StoredIcon icon={watchedIcon} fallback={getTypeIcon(watchedType)} className="h-5 w-5" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">{watchedName?.trim() || "Tool name"}</span>
                  <span className="block truncate text-xs text-muted-foreground">
                    {typeOption?.label} • v{watch("version") || "1.0.0"}
                  </span>
                </span>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="icon">Icon</Label>
                <IconPicker
                  id="icon"
                  value={watchedIcon ?? ""}
                  onChange={(value) => setValue("icon", value, { shouldDirty: true, shouldValidate: true })}
                  invalid={!!errors.icon}
                  disabled={isFormLoading}
                />
                <FieldError message={errors.icon?.message} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <ColorField
                  id="color"
                  label="Icon colour"
                  value={watchedColor}
                  fallback={previewColor}
                  disabled={isFormLoading}
                  onChange={(value) => setValue("color", value.toUpperCase(), { shouldDirty: true })}
                  inputProps={register("color")}
                />
                <ColorField
                  id="backgroundColor"
                  label="Tile colour"
                  value={watchedBackground}
                  fallback={previewColor}
                  disabled={isFormLoading}
                  onChange={(value) => setValue("backgroundColor", value.toUpperCase(), { shouldDirty: true })}
                  inputProps={register("backgroundColor")}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Version</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="version">Version label *</Label>
                <Input
                  id="version"
                  {...register("version")}
                  placeholder="1.0.0"
                  className={cn("font-mono", errors.version && "border-destructive")}
                  disabled={isFormLoading}
                />
                <FieldError message={errors.version?.message} />
                <p className="text-xs text-muted-foreground">Replaced by the published version number when a version is published.</p>
              </div>
              <div className="flex gap-2 rounded-md border border-dashed p-3 text-xs text-muted-foreground">
                <FileCode2 className="mt-0.5 h-4 w-4 shrink-0" />
                <p>
                  The tool&apos;s logic is written after saving, on its authoring page, then reviewed and published.
                  {appFile ? <span className="mt-1 block font-mono">Legacy artifact: {appFile}</span> : null}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="flex items-center justify-end gap-3 border-t pt-6">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isFormLoading}>
          Cancel
        </Button>
        <Button type="submit" disabled={isFormLoading}>
          {isFormLoading && (
            <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
          )}
          {mode === "create" ? "Create Tool" : "Update Tool"}
        </Button>
      </div>
    </form>
  )
}
