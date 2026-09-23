"use client"

import * as React from "react"
import { Loader2, Plus } from "lucide-react"
import { GuidelineUploadProgress } from "./guideline-upload-progress"
import type { UploadOptions } from "@/services/guideline-upload.service"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { FileUpload } from "@/components/ui/file-upload"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  AS_UPLOADED_SOURCE_ACCEPT,
  CreateGuidelineVersionInput,
  GUIDELINE_SOURCE_ACCEPT,
  GuidelineDocumentRecord,
  GuidelineVersionRecord,
  IngestionJobRecord,
} from "@/services/guideline-documents.service"

export function CreateVersionDialog({
  document,
  open,
  submitting,
  onOpenChange,
  onSubmit,
}: {
  document: GuidelineDocumentRecord | null
  open: boolean
  submitting: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: CreateGuidelineVersionInput) => Promise<void>
}) {
  const [version, setVersion] = React.useState("")
  const [publicationDate, setPublicationDate] = React.useState("")
  const [reviewDate, setReviewDate] = React.useState("")

  React.useEffect(() => {
    if (!open) {
      setVersion("")
      setPublicationDate("")
      setReviewDate("")
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Guideline Version</DialogTitle>
          <DialogDescription>
            Add a version record for {document?.title || "this guideline"} before uploading PDF or Markdown.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="version-number">Version</Label>
            <Input
              id="version-number"
              placeholder="2026.1"
              value={version}
              onChange={(event) => setVersion(event.target.value)}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="publication-date">Publication Date</Label>
              <Input
                id="publication-date"
                type="date"
                value={publicationDate}
                onChange={(event) => setPublicationDate(event.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="review-date">Review Date</Label>
              <Input
                id="review-date"
                type="date"
                value={reviewDate}
                onChange={(event) => setReviewDate(event.target.value)}
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <Button
            onClick={() =>
              onSubmit({
                version: version.trim(),
                publication_date: publicationDate || undefined,
                review_date: reviewDate || undefined,
              })
            }
            disabled={submitting || !version.trim()}
          >
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            Create Version
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function UploadVersionDialog({
  version,
  asUploaded = false,
  open,
  submitting,
  onOpenChange,
  onSubmit,
}: {
  version: GuidelineVersionRecord | null
  /** Forms and other kinds published exactly as uploaded take PDF or Word files. */
  asUploaded?: boolean
  open: boolean
  submitting: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (file: File, options?: UploadOptions) => Promise<IngestionJobRecord | void>
}) {
  const [file, setFile] = React.useState<File | null>(null)

  React.useEffect(() => {
    if (!open) {
      setFile(null)
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={(value) => { if (!submitting) onOpenChange(value) }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{asUploaded ? "Upload Form File" : "Upload Guideline Source"}</DialogTitle>
          <DialogDescription>
            {asUploaded
              ? `Upload a PDF or Word (.docx) file for version ${version?.version || ""}. It is kept exactly as uploaded; its text is only indexed for search.`
              : `Upload PDF or UTF-8 Markdown for version ${version?.version || ""}. The backend will extract structured content and rebuild the AI index.`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label>{asUploaded ? "PDF or Word File" : "PDF or Markdown File"}</Label>
          <FileUpload
            value={file || undefined}
            onValueChange={(next) => { if (!submitting) setFile(next) }}
            accept={asUploaded ? AS_UPLOADED_SOURCE_ACCEPT : GUIDELINE_SOURCE_ACCEPT}
            maxSize={100}
            placeholder={asUploaded ? "Choose form PDF or Word file" : "Choose guideline PDF or Markdown file"}
          />
        </div>

        {version && <GuidelineUploadProgress key={version.id} versionId={version.id} file={file} submitting={submitting} onSubmit={onSubmit} />}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
