"use client"

import * as React from "react"
import { Link2, Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { showToast } from "@/lib/toast"
import {
  guidelineLinkError,
  GuidelineDocumentsService,
  GuidelineVersionRecord,
} from "@/services/guideline-documents.service"

/**
 * Saves the https link a link version publishes. The link is checked in the
 * browser before it is sent, and again by the backend.
 */
export function GuidelineLinkForm({
  version,
  onSaved,
}: {
  version: GuidelineVersionRecord
  onSaved?: (version: GuidelineVersionRecord) => void | Promise<void>
}) {
  const [value, setValue] = React.useState(version.external_url || "")
  const [touched, setTouched] = React.useState(false)
  const [saving, setSaving] = React.useState(false)
  const error = guidelineLinkError(value)
  const showError = touched && error
  const unchanged = value.trim() === (version.external_url || "")

  async function save(event: React.FormEvent) {
    event.preventDefault()
    setTouched(true)
    if (error) return
    setSaving(true)
    try {
      const saved = await GuidelineDocumentsService.setVersionLink(version.id, value)
      showToast.success("Link saved", "Readers will open this link when the version is published.")
      await onSaved?.(saved)
    } catch (failure) {
      showToast.error("Link not saved", failure instanceof Error ? failure.message : "Unknown error")
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="space-y-2" onSubmit={save} noValidate>
      <Label htmlFor={`guideline-link-${version.id}`}>Link</Label>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          id={`guideline-link-${version.id}`}
          type="url"
          inputMode="url"
          autoComplete="url"
          placeholder="https://www.who.int/publications/..."
          value={value}
          maxLength={2048}
          aria-invalid={Boolean(showError)}
          aria-describedby={`guideline-link-${version.id}-hint`}
          onChange={(event) => setValue(event.target.value)}
          onBlur={() => setTouched(true)}
          disabled={saving}
        />
        <Button type="submit" disabled={saving || (touched && Boolean(error)) || (unchanged && Boolean(version.external_url))}>
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Link2 className="h-4 w-4" />}
          {version.external_url ? "Update link" : "Save link"}
        </Button>
      </div>
      <p
        id={`guideline-link-${version.id}-hint`}
        className={showError ? "text-sm text-destructive" : "text-xs text-muted-foreground"}
      >
        {showError ? error : "A full link starting with https://. Readers open it directly; nothing is uploaded or extracted."}
      </p>
    </form>
  )
}
