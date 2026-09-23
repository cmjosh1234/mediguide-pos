"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, FileText, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  GuidelineDocumentsService,
  originalFileName,
} from "@/services/guideline-documents.service";

/**
 * Shows a stored original file exactly as uploaded: PDFs inline, Word files as
 * a download (browsers cannot render .docx).
 */
export function OriginalFilePreview({
  versionId,
  fileKey,
  className = "h-[70vh]",
}: {
  versionId: string;
  fileKey?: string | null;
  className?: string;
}) {
  const fileQuery = useQuery({
    queryKey: ["guideline-original-file", versionId, fileKey],
    queryFn: () => GuidelineDocumentsService.getOriginalPdf(versionId),
    enabled: Boolean(fileKey),
    staleTime: Infinity,
  });
  const [url, setUrl] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!fileQuery.data) return;
    const objectUrl = URL.createObjectURL(fileQuery.data);
    setUrl(objectUrl);
    return () => URL.revokeObjectURL(objectUrl);
  }, [fileQuery.data]);

  const name = originalFileName(fileKey) || "original file";
  const isPdf =
    fileQuery.data?.type === "application/pdf" ||
    name.toLowerCase().endsWith(".pdf");

  if (!fileKey) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
        No file has been uploaded for this version yet.
      </div>
    );
  }
  if (fileQuery.isLoading || (fileQuery.data && !url)) {
    return (
      <div className="flex items-center justify-center gap-2 rounded-lg border p-8 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading {name}…
      </div>
    );
  }
  if (fileQuery.isError || !url) {
    return (
      <div className="rounded-lg border border-destructive/40 p-6 text-sm text-destructive">
        The stored file could not be loaded.
      </div>
    );
  }
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2 text-sm">
          <FileText className="h-4 w-4 shrink-0" />
          <span className="truncate font-medium">{name}</span>
        </div>
        <Button variant="outline" size="sm" asChild>
          <a href={url} download={name}>
            <Download className="h-4 w-4" /> Download original
          </a>
        </Button>
      </div>
      {isPdf ? (
        <iframe
          title={`Preview of ${name}`}
          src={url}
          className={`w-full rounded-lg border bg-muted ${className}`}
        />
      ) : (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          Word documents cannot be previewed in the browser. Download the file
          to see it exactly as it was uploaded.
        </div>
      )}
    </div>
  );
}
