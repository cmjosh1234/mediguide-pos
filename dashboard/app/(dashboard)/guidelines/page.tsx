"use client"

import * as React from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import { useRouter, useSearchParams } from "next/navigation"

import { ConfirmDialog } from "@/components/dialogs/confirm-dialog"
import { Badge } from "@/components/ui/badge"
import { DataTable } from "@/components/ui/data-table"
import { LoadingState } from "@/components/ui/loading-state"
import { PageHeader } from "@/components/ui/page-header"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { usePermissionContext } from "@/lib/permission-context"
import { hasBackendPermission } from "@/lib/backend-client"
import { showToast } from "@/lib/toast"
import {
  CreateGuidelineVersionInput,
  getDocumentLatestVersion,
  GuidelineDocumentRecord,
  GuidelineDocumentsService,
  GuidelineVersionRecord,
  guidelineDocumentsQueryKey,
  isPublishedAsUploaded,
} from "@/services/guideline-documents.service"
import { documentKindService, documentKindsQueryKey } from "@/services/document-kinds.service"
import { createGuidelinesColumns } from "./columns"
import {
  CreateVersionDialog,
  UploadVersionDialog,
} from "./components/guideline-version-dialogs"
import { GuidelineNotificationDialog } from "./components/guideline-notification-dialog"

const ALL_KINDS = "all"

export default function GuidelinesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const queryClient = useQueryClient()
  const { hasPermission, loading: permissionsLoading } = usePermissionContext()
  const canUpdate = hasPermission("content", "update:any")
  const canNotify = hasBackendPermission("notification.campaign.manage")
  // Mirrors the DELETE /api/v2/guidelines/:id route guard.
  const canDelete = hasBackendPermission("guideline.write") && hasBackendPermission("guideline.publish")
  const [versionDocument, setVersionDocument] = React.useState<GuidelineDocumentRecord | null>(null)
  const [uploadVersion, setUploadVersion] = React.useState<GuidelineVersionRecord | null>(null)
  const [notificationDocument, setNotificationDocument] = React.useState<GuidelineDocumentRecord | null>(null)
  const [deletingDocument, setDeletingDocument] = React.useState<GuidelineDocumentRecord | null>(null)
  const [submitting, setSubmitting] = React.useState(false)

  React.useEffect(() => {
    if (!permissionsLoading && !hasPermission("content", "read:any")) router.replace("/")
  }, [hasPermission, permissionsLoading, router])

  const documentsQuery = useQuery({
    queryKey: guidelineDocumentsQueryKey,
    queryFn: () => GuidelineDocumentsService.listDocuments(),
  })
  const kindsQuery = useQuery({
    queryKey: documentKindsQueryKey,
    queryFn: () => documentKindService.list(),
  })

  const documents = React.useMemo(() => documentsQuery.data?.items || [], [documentsQuery.data])
  // One filter option per active kind, plus any inactive kind that still has documents
  // so no document is unreachable from the filter.
  const kindOptions = React.useMemo(() => {
    const counts = new Map<string, number>()
    for (const document of documents) {
      if (document.document_kind_id) {
        counts.set(document.document_kind_id, (counts.get(document.document_kind_id) || 0) + 1)
      }
    }
    return (kindsQuery.data || [])
      .filter((kind) => kind.status === "active" || counts.has(kind.id))
      .map((kind) => ({ ...kind, count: counts.get(kind.id) || 0 }))
  }, [documents, kindsQuery.data])
  const selectedKind = kindOptions.find((kind) => kind.slug === searchParams.get("kind"))
  const visibleDocuments = React.useMemo(
    () => (selectedKind ? documents.filter((document) => document.document_kind_id === selectedKind.id) : documents),
    [documents, selectedKind]
  )

  function selectKind(slug: string) {
    router.replace(slug === ALL_KINDS ? "/guidelines" : `/guidelines?kind=${encodeURIComponent(slug)}`, { scroll: false })
  }

  const refresh = React.useCallback(
    () => queryClient.invalidateQueries({ queryKey: guidelineDocumentsQueryKey }),
    [queryClient]
  )

  const columns = React.useMemo(
    () =>
      createGuidelinesColumns({
        canUpdate,
        canNotify,
        canDelete,
        onView: (document) => router.push(`/guidelines/${document.id}`),
        onEdit: (document) => router.push(`/guidelines/${document.id}/edit`),
        onNewVersion: setVersionDocument,
        onUpload: (document) => {
          const version = getDocumentLatestVersion(document)
          if (version) setUploadVersion(version)
        },
        onNotify: setNotificationDocument,
        onDelete: setDeletingDocument,
      }),
    [canDelete, canNotify, canUpdate, router]
  )

  async function createVersion(payload: CreateGuidelineVersionInput) {
    if (!versionDocument) return
    setSubmitting(true)
    try {
      await GuidelineDocumentsService.createVersion(versionDocument.id, payload)
      setVersionDocument(null)
      await refresh()
      showToast.success("Version created", "The new version is ready for PDF or Markdown upload.")
    } catch (error) {
      showToast.error("Create version failed", error instanceof Error ? error.message : "Unknown error")
    } finally {
      setSubmitting(false)
    }
  }

  async function uploadSource(file: File, options?: import("@/services/guideline-upload.service").UploadOptions) {
    if (!uploadVersion) return
    setSubmitting(true)
    try {
      const job = await GuidelineDocumentsService.uploadVersionSource(uploadVersion.id, file, options)
      await refresh()
      const asUploaded = isPublishedAsUploaded(documents.find((document) => document.id === uploadVersion.document_id))
      showToast.success(
        asUploaded ? "Form stored" : "Source uploaded",
        asUploaded
          ? "The file is kept exactly as uploaded. Its text is being indexed for search."
          : "Document extraction and indexing have been queued."
      )
      return job
    } catch (error) {
      throw error
    } finally {
      setSubmitting(false)
    }
  }

  async function deleteGuideline() {
    if (!deletingDocument) return
    setSubmitting(true)
    try {
      await GuidelineDocumentsService.deleteDocument(deletingDocument.id)
      await refresh()
      showToast.success("Guideline deleted", `"${deletingDocument.title}" is no longer available to users.`)
    } catch (error) {
      showToast.error("Delete failed", error instanceof Error ? error.message : "Unknown error")
    } finally {
      setSubmitting(false)
    }
  }

  if (permissionsLoading || documentsQuery.isLoading) {
    return <LoadingState message="Loading guideline documents..." />
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Medical Guidelines"
        description="Manage v2 guideline documents, extracted assets, and publication versions."
        actions={
          hasPermission("content", "create:any")
            ? [{
                label: "Create Guideline",
                onClick: () => router.push("/guidelines/create"),
                icon: <Plus className="h-4 w-4" />,
              }]
            : []
        }
      />

      {documentsQuery.isError ? (
        <div className="rounded-md border border-destructive p-4 text-sm text-destructive">
          {documentsQuery.error instanceof Error
            ? documentsQuery.error.message
            : "Failed to load guideline documents"}
        </div>
      ) : (
        <div className="space-y-4">
          {kindOptions.length > 0 ? (
            <div className="flex flex-col gap-1.5 sm:w-72">
              <Label htmlFor="guideline-kind-filter">Document kind</Label>
              <Select value={selectedKind?.slug ?? ALL_KINDS} onValueChange={selectKind}>
                <SelectTrigger id="guideline-kind-filter" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL_KINDS}>
                    All
                    <Badge variant="secondary" className="px-1.5 tabular-nums">{documents.length}</Badge>
                  </SelectItem>
                  {kindOptions.map((kind) => (
                    <SelectItem key={kind.id} value={kind.slug}>
                      {kind.name}
                      <Badge variant="secondary" className="px-1.5 tabular-nums">{kind.count}</Badge>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}
          <DataTable
            columns={columns}
            data={visibleDocuments}
            searchKey="title"
            searchPlaceholder="Search guideline titles..."
            tableClassName="min-w-[1380px]"
          />
        </div>
      )}

      <CreateVersionDialog
        document={versionDocument}
        open={Boolean(versionDocument)}
        submitting={submitting}
        onOpenChange={(open) => !open && setVersionDocument(null)}
        onSubmit={createVersion}
      />
      <UploadVersionDialog
        version={uploadVersion}
        asUploaded={isPublishedAsUploaded(documents.find((document) => document.id === uploadVersion?.document_id))}
        open={Boolean(uploadVersion)}
        submitting={submitting}
        onOpenChange={(open) => !open && setUploadVersion(null)}
        onSubmit={uploadSource}
      />
      <GuidelineNotificationDialog
        document={notificationDocument}
        open={Boolean(notificationDocument)}
        onOpenChange={(open) => !open && setNotificationDocument(null)}
      />
      <ConfirmDialog
        open={Boolean(deletingDocument)}
        onOpenChange={(open) => !open && setDeletingDocument(null)}
        title="Delete guideline"
        description={`Delete "${deletingDocument?.title ?? ""}"? It will be removed from the public library, the mobile app and search immediately. Version history is kept for audit.`}
        confirmText="Delete"
        variant="destructive"
        onConfirm={deleteGuideline}
        loading={submitting}
      />
    </div>
  )
}
