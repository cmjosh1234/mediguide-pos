"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type {
  DocumentKind,
  DocumentKindInput,
} from "@/services/document-kinds.service";

const emptyKind: DocumentKindInput = {
  name: "",
  slug: "",
  description: "",
  sort_order: 0,
  status: "active",
  publish_as_uploaded: false,
};

export function DocumentKindDialog({
  open,
  kind,
  submitting,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  /** The kind being edited, or null to create a new one. */
  kind: DocumentKind | null;
  submitting: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (value: DocumentKindInput) => Promise<void>;
}) {
  const [value, setValue] = React.useState<DocumentKindInput>(emptyKind);

  React.useEffect(() => {
    if (!open) return;
    setValue(
      kind
        ? {
            name: kind.name,
            slug: kind.slug,
            description: kind.description,
            sort_order: kind.sort_order,
            status: kind.status,
            publish_as_uploaded: kind.publish_as_uploaded,
          }
        : emptyKind,
    );
  }, [kind, open]);

  const sortOrder = value.sort_order ?? 0;
  const valid =
    value.name.trim().length > 0 &&
    Number.isInteger(sortOrder) &&
    sortOrder >= 0 &&
    sortOrder <= 10000;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            if (valid) void onSubmit(value);
          }}
        >
          <DialogHeader>
            <DialogTitle>{kind ? "Edit document kind" : "Add document kind"}</DialogTitle>
            <DialogDescription>
              Document kinds classify guideline and outbreak documents, for
              example guidelines, forms and SOPs. Each kind appears as a tab on
              the guidelines list.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="document-kind-name">Name</Label>
            <Input
              id="document-kind-name"
              value={value.name}
              maxLength={120}
              placeholder="Form"
              onChange={(event) => setValue((current) => ({ ...current, name: event.target.value }))}
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="document-kind-slug">Slug</Label>
            <Input
              id="document-kind-slug"
              value={value.slug || ""}
              placeholder="Generated from the name"
              readOnly={Boolean(kind)}
              className={kind ? "bg-muted font-mono" : "font-mono"}
              onChange={(event) => setValue((current) => ({ ...current, slug: event.target.value }))}
            />
            <p className="text-xs text-muted-foreground">
              {kind
                ? "The slug is a permanent code used by documents and the mobile app, so it cannot be changed."
                : "Lowercase letters, numbers, underscores or hyphens, for example contact_tracing_guide. It cannot be changed later."}
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="document-kind-description">Description</Label>
            <Textarea
              id="document-kind-description"
              rows={3}
              value={value.description || ""}
              onChange={(event) => setValue((current) => ({ ...current, description: event.target.value }))}
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="document-kind-sort-order">Sort order</Label>
              <Input
                id="document-kind-sort-order"
                type="number"
                min={0}
                max={10000}
                step={1}
                value={sortOrder}
                onChange={(event) =>
                  setValue((current) => ({ ...current, sort_order: event.target.valueAsNumber || 0 }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="document-kind-active">Active</Label>
              <div className="flex h-10 items-center gap-2">
                <Switch
                  id="document-kind-active"
                  checked={value.status === "active"}
                  onCheckedChange={(checked) =>
                    setValue((current) => ({ ...current, status: checked ? "active" : "inactive" }))
                  }
                />
                <span className="text-sm text-muted-foreground">
                  {value.status === "active" ? "Can be assigned" : "Hidden from new documents"}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-start justify-between gap-4 rounded-lg border p-3">
            <div className="space-y-1">
              <Label htmlFor="document-kind-as-uploaded">Publish as uploaded file</Label>
              <p className="text-xs text-muted-foreground">
                {kind && kind.guideline_document_count > 0
                  ? `Locked while ${kind.guideline_document_count} guideline document(s) use this kind.`
                  : "For forms: the PDF or Word file is published exactly as uploaded. Its text is indexed for search, and it never goes through the Markdown editor."}
              </p>
            </div>
            <Switch
              id="document-kind-as-uploaded"
              checked={Boolean(value.publish_as_uploaded)}
              disabled={Boolean(kind && kind.guideline_document_count > 0)}
              onCheckedChange={(checked) =>
                setValue((current) => ({ ...current, publish_as_uploaded: checked }))
              }
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" disabled={submitting} onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting || !valid}>
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {kind ? "Save changes" : "Add kind"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
