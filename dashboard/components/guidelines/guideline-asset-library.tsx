"use client"
/* eslint-disable @next/next/no-img-element -- asset URLs are short-lived signed URLs with dynamic hosts */

import * as React from "react"
import { CheckCircle2, CircleAlert, CopyCheck, ImagePlus, Pencil, RefreshCw, Search, Trash2, UploadCloud, X } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Progress } from "@/components/ui/progress"
import { ScrollArea } from "@/components/ui/scroll-area"
import { showToast } from "@/lib/toast"
import { GuidelineAsset, GuidelineAssetMetadata, GuidelineAssetsService } from "@/services/guideline-assets.service"
import { assetMatchesLibraryFilter, guidelineFileChecksum, type GuidelineAssetLibraryFilter, validateGuidelineImageFile } from "./guideline-asset-library-utils"

interface GuidelineAssetLibraryProps {
  open: boolean
  versionId: string
  editable: boolean
  onOpenChange: (open: boolean) => void
  onInsert: (asset: GuidelineAsset) => void
  onAssetsChange: (assets: GuidelineAsset[]) => void
  onReplaceReference: (oldReference: string, replacement: GuidelineAsset) => void
}

type QueueStatus = "checking" | "ready" | "duplicate" | "uploading" | "uploaded" | "failed"

interface QueuedImage {
  id: string
  file: File
  checksum?: string
  duplicateName?: string
  metadata: GuidelineAssetMetadata
  status: QueueStatus
  progress: number
  error?: string
}

const emptyMetadata: GuidelineAssetMetadata = { alternative_text: "", caption: "", source: "", attribution: "", license: "", clinically_sensitive: false }
const filterLabels: Record<GuidelineAssetLibraryFilter, string> = {
  all: "All images", used: "Used", unused: "Unused", pending: "Pending review",
  reviewed: "Reviewed", rejected: "Rejected", broken: "Broken references",
}

export function GuidelineAssetLibrary({ open, versionId, editable, onOpenChange, onInsert, onAssetsChange, onReplaceReference }: GuidelineAssetLibraryProps) {
  const [assets, setAssets] = React.useState<GuidelineAsset[]>([])
  const [broken, setBroken] = React.useState<string[]>([])
  const [loading, setLoading] = React.useState(false)
  const [queue, setQueue] = React.useState<QueuedImage[]>([])
  const [uploadingQueue, setUploadingQueue] = React.useState(false)
  const [replacementFile, setReplacementFile] = React.useState<File | null>(null)
  const [replacing, setReplacing] = React.useState<GuidelineAsset | null>(null)
  const [editing, setEditing] = React.useState<GuidelineAsset | null>(null)
  const [metadata, setMetadata] = React.useState<GuidelineAssetMetadata>(emptyMetadata)
  const [query, setQuery] = React.useState("")
  const [filter, setFilter] = React.useState<GuidelineAssetLibraryFilter>("all")
  const bulkInputRef = React.useRef<HTMLInputElement>(null)
  const replacementInputRef = React.useRef<HTMLInputElement>(null)

  const load = React.useCallback(async () => {
    setLoading(true)
    try {
      const result = await GuidelineAssetsService.list(versionId)
      setAssets(result.items)
      setBroken(result.broken_references)
      onAssetsChange(result.items)
    } catch (error) {
      showToast.error("Assets unavailable", error instanceof Error ? error.message : "Could not load guideline assets")
    } finally { setLoading(false) }
  }, [onAssetsChange, versionId])

  React.useEffect(() => { if (open) void load() }, [load, open])

  const updateQueue = React.useCallback((id: string, values: Partial<QueuedImage>) => {
    setQueue((current) => current.map((item) => item.id === id ? { ...item, ...values } : item))
  }, [])

  const enqueue = React.useCallback(async (files: File[]) => {
    if (!files.length) return
    const incoming = files.map((file, index) => ({
      id: `${Date.now()}-${index}-${file.name}`,
      file,
      metadata: { ...emptyMetadata },
      status: "checking" as const,
      progress: 5,
      error: validateGuidelineImageFile(file) || undefined,
    }))
    setQueue((current) => [...current, ...incoming])
    const known = new Map(assets.map((asset) => [asset.checksum.toLowerCase(), asset.original_filename]))
    for (const item of incoming) {
      if (item.error) { updateQueue(item.id, { status: "failed", progress: 0 }); continue }
      try {
        const checksum = await guidelineFileChecksum(item.file)
        const duplicateName = known.get(checksum)
        if (duplicateName) updateQueue(item.id, { checksum, duplicateName, status: "duplicate", progress: 100 })
        else updateQueue(item.id, { checksum, status: "ready", progress: 0 })
      } catch { updateQueue(item.id, { status: "failed", progress: 0, error: "Could not calculate the file checksum." }) }
    }
  }, [assets, updateQueue])

  const chooseReplacement = (file: File, asset: GuidelineAsset) => {
    const error = validateGuidelineImageFile(file)
    if (error) { showToast.error("Unsupported image", error); return }
    setReplacementFile(file)
    setReplacing(asset)
    setMetadata(metadataFrom(asset))
  }

  const uploadOne = React.useCallback(async (item: QueuedImage) => {
    const fileError = validateGuidelineImageFile(item.file)
    if (fileError) {
      updateQueue(item.id, { status: "failed", progress: 0, error: fileError })
      return false
    }
    if (!item.metadata.alternative_text?.trim()) {
      updateQueue(item.id, { status: "failed", error: "Alternative text is required." })
      return false
    }
    updateQueue(item.id, { status: "uploading", progress: 20, error: undefined })
    try {
      await GuidelineAssetsService.upload(versionId, item.file, item.metadata)
      updateQueue(item.id, { status: "uploaded", progress: 100 })
      return true
    } catch (error) {
      updateQueue(item.id, { status: "failed", progress: 0, error: error instanceof Error ? error.message : "Could not upload image" })
      return false
    }
  }, [updateQueue, versionId])

  const uploadAll = async () => {
    const candidates = queue.filter((item) =>
      (item.status === "ready" || item.status === "failed") && !validateGuidelineImageFile(item.file),
    )
    if (!candidates.length) return
    const missingAlt = candidates.filter((item) => !item.metadata.alternative_text?.trim())
    if (missingAlt.length) {
      showToast.warning("Alternative text required", `Add alternative text to ${missingAlt.length} queued image${missingAlt.length === 1 ? "" : "s"}.`)
      return
    }
    setUploadingQueue(true)
    let uploaded = 0
    for (const item of candidates) if (await uploadOne(item)) uploaded += 1
    await load()
    setUploadingQueue(false)
    if (uploaded) showToast.success("Images uploaded", `${uploaded} image${uploaded === 1 ? " is" : "s are"} ready to insert.`)
  }

  const uploadReplacement = async () => {
    if (!replacementFile || !replacing) return
    try {
      const created = await GuidelineAssetsService.upload(versionId, replacementFile, metadata)
      onReplaceReference(replacing.reference, created)
      await load()
      setReplacementFile(null); setReplacing(null)
      showToast.success("Replacement uploaded", "The Markdown reference was updated; the old asset remains available until explicitly removed.")
    } catch (error) { showToast.error("Upload failed", error instanceof Error ? error.message : "Could not upload image") }
  }

  const saveMetadata = async () => {
    if (!editing) return
    try {
      await GuidelineAssetsService.update(versionId, editing.id, metadata)
      setEditing(null); await load()
      showToast.success("Asset updated", "Metadata changes require review before publication.")
    } catch (error) { showToast.error("Update failed", error instanceof Error ? error.message : "Could not update asset") }
  }

  const remove = async (asset: GuidelineAsset) => {
    if (asset.referenced || !window.confirm(`Remove ${asset.original_filename}? The stored object is retained for immutable historical revisions.`)) return
    try { await GuidelineAssetsService.remove(versionId, asset.id); await load(); showToast.success("Asset removed", "The asset is no longer available to this draft.") }
    catch (error) { showToast.error("Removal failed", error instanceof Error ? error.message : "Could not remove asset") }
  }

  const filteredAssets = React.useMemo(() => assets.filter((asset) => assetMatchesLibraryFilter(asset, query, filter)), [assets, filter, query])
  const queueComplete = queue.filter((item) => item.status === "uploaded" || item.status === "duplicate").length
  const queueProgress = queue.length ? Math.round((queueComplete / queue.length) * 100) : 0

  return <>
    <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-w-6xl"><DialogHeader><DialogTitle>Guideline asset library</DialogTitle><DialogDescription>Upload a batch, complete its metadata, then insert each version-scoped image where it belongs.</DialogDescription></DialogHeader>
      {broken.length > 0 && filter !== "broken" && <Alert variant="destructive"><AlertTitle>Broken image references</AlertTitle><AlertDescription>{broken.length} Markdown reference{broken.length === 1 ? " does" : "s do"} not resolve to an image in this version. Select the Broken references filter to inspect them.</AlertDescription></Alert>}
      {editable && <div tabIndex={0} className="rounded-lg border-2 border-dashed p-6 text-center" onDragOver={(event) => event.preventDefault()} onDrop={(event) => { event.preventDefault(); void enqueue([...event.dataTransfer.files]) }} onPaste={(event) => {
        const files = [...event.clipboardData.items].filter((item) => item.type.startsWith("image/")).map((item, index) => { const file = item.getAsFile(); return file ? new File([file], file.name || `clipboard-${Date.now()}-${index}.png`, { type: file.type }) : null }).filter((file): file is File => file !== null)
        void enqueue(files)
      }}><ImagePlus className="mx-auto h-7 w-7" /><p className="mt-2 text-sm">Drop or paste multiple images here, or choose files.</p><p className="mt-1 text-xs text-muted-foreground">PNG, JPEG, GIF or WebP · up to 10 MB each</p><Button className="mt-3" variant="outline" onClick={() => bulkInputRef.current?.click()}><UploadCloud className="mr-2 h-4 w-4" />Choose images</Button><input ref={bulkInputRef} className="hidden" type="file" multiple accept="image/png,image/jpeg,image/gif,image/webp" onChange={(event) => { void enqueue([...event.target.files || []]); event.target.value = "" }} /><input ref={replacementInputRef} className="hidden" type="file" accept="image/png,image/jpeg,image/gif,image/webp" onChange={(event) => { const file = event.target.files?.[0]; if (file && replacing) chooseReplacement(file, replacing); event.target.value = "" }} /></div>}
      {queue.length > 0 && <UploadQueue items={queue} uploading={uploadingQueue} progress={queueProgress} onMetadata={(id, next) => setQueue((current) => current.map((item) => item.id === id ? { ...item, metadata: next, error: item.error === "Alternative text is required." ? undefined : item.error, status: item.status === "failed" && item.error === "Alternative text is required." ? "ready" : item.status } : item))} onRemove={(id) => setQueue((current) => current.filter((item) => item.id !== id))} onRetry={(item) => void uploadOne(item).then((success) => { if (success) void load() })} onUploadAll={() => void uploadAll()} onClearCompleted={() => setQueue((current) => current.filter((item) => item.status !== "uploaded" && item.status !== "duplicate"))} />}
      <div className="flex flex-col gap-2 sm:flex-row"><div className="relative flex-1"><Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" /><Input className="pl-9" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search filename, alternative text, caption or source" /></div><select aria-label="Filter guideline assets" className="h-10 rounded-md border bg-background px-3 text-sm" value={filter} onChange={(event) => setFilter(event.target.value as GuidelineAssetLibraryFilter)}>{Object.entries(filterLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></div>
      <ScrollArea className="max-h-[48vh]">{filter === "broken" ? <BrokenReferences references={broken} /> : <div className="grid gap-3 pr-3 sm:grid-cols-2">{loading && <p className="text-sm text-muted-foreground">Loading assets…</p>}{!loading && filteredAssets.length === 0 && <p className="col-span-full py-8 text-center text-sm text-muted-foreground">No images match this search and filter.</p>}{filteredAssets.map((asset) => <AssetCard key={asset.id} asset={asset} editable={editable} onInsert={onInsert} onEdit={() => { setEditing(asset); setMetadata(metadataFrom(asset)) }} onReplace={() => { setReplacing(asset); replacementInputRef.current?.click() }} onRemove={() => void remove(asset)} />)}</div>}</ScrollArea>
    </DialogContent></Dialog>
    <Dialog open={Boolean(replacementFile)} onOpenChange={(value) => { if (!value) { setReplacementFile(null); setReplacing(null) } }}><DialogContent><DialogHeader><DialogTitle>Upload replacement image</DialogTitle><DialogDescription>Alternative text is required. Clinically sensitive figures require reviewer approval.</DialogDescription></DialogHeader><AssetMetadataFields idPrefix="replacement" metadata={metadata} setMetadata={setMetadata} /><DialogFooter><Button variant="outline" onClick={() => { setReplacementFile(null); setReplacing(null) }}>Cancel</Button><Button disabled={!metadata.alternative_text?.trim()} onClick={() => void uploadReplacement()}>Upload replacement</Button></DialogFooter></DialogContent></Dialog>
    <Dialog open={Boolean(editing)} onOpenChange={(value) => { if (!value) setEditing(null) }}><DialogContent><DialogHeader><DialogTitle>Edit asset metadata</DialogTitle><DialogDescription>The underlying file and immutable checksum are unchanged.</DialogDescription></DialogHeader><AssetMetadataFields idPrefix="edit" metadata={metadata} setMetadata={setMetadata} /><DialogFooter><Button variant="outline" onClick={() => setEditing(null)}>Cancel</Button><Button disabled={!metadata.alternative_text?.trim()} onClick={() => void saveMetadata()}>Save metadata</Button></DialogFooter></DialogContent></Dialog>
  </>
}

function UploadQueue({ items, uploading, progress, onMetadata, onRemove, onRetry, onUploadAll, onClearCompleted }: { items: QueuedImage[]; uploading: boolean; progress: number; onMetadata: (id: string, metadata: GuidelineAssetMetadata) => void; onRemove: (id: string) => void; onRetry: (item: QueuedImage) => void; onUploadAll: () => void; onClearCompleted: () => void }) {
  const actionable = items.filter((item) =>
    (item.status === "ready" || item.status === "failed") && !validateGuidelineImageFile(item.file),
  )
  const complete = items.filter((item) => item.status === "uploaded" || item.status === "duplicate").length
  return <section className="space-y-3 rounded-lg border p-3" aria-label="Image upload queue"><div className="flex flex-wrap items-center justify-between gap-2"><div><h3 className="font-medium">Upload queue</h3><p className="text-xs text-muted-foreground">{complete} of {items.length} complete or already present</p></div><div className="flex gap-2"><Button size="sm" variant="ghost" disabled={!complete || uploading} onClick={onClearCompleted}>Clear completed</Button><Button size="sm" disabled={!actionable.length || uploading} onClick={onUploadAll}>{uploading ? "Uploading…" : `Upload ${actionable.length || "all"}`}</Button></div></div><Progress value={progress} aria-label={`${progress}% of queued images complete`} /><ScrollArea className="max-h-[34vh]"><div className="space-y-3 pr-3">{items.map((item) => <article key={item.id} className="rounded-md border p-3"><div className="flex items-start justify-between gap-3"><div className="min-w-0"><p className="truncate text-sm font-medium" title={item.file.name}>{item.file.name}</p><p className="text-xs text-muted-foreground">{(item.file.size / 1024).toFixed(1)} KB</p></div><div className="flex items-center gap-2"><QueueStatusBadge item={item} />{item.status !== "uploading" && <Button size="icon" variant="ghost" aria-label={`Remove ${item.file.name} from queue`} onClick={() => onRemove(item.id)}><X className="h-4 w-4" /></Button>}</div></div>{item.status !== "duplicate" && item.status !== "uploaded" && <div className="mt-3"><AssetMetadataFields compact idPrefix={`queue-${item.id}`} metadata={item.metadata} setMetadata={(value) => onMetadata(item.id, value)} /></div>}{item.status === "uploading" && <Progress className="mt-3" value={item.progress} aria-label={`Uploading ${item.file.name}`} />}{item.status === "duplicate" && <p className="mt-2 text-xs text-muted-foreground">Identical to {item.duplicateName}. The existing governed asset was kept.</p>}{item.error && <div className="mt-2 flex items-center justify-between gap-2 text-xs text-destructive"><span>{item.error}</span>{item.metadata.alternative_text?.trim() && <Button size="sm" variant="outline" onClick={() => onRetry(item)}>Retry</Button>}</div>}</article>)}</div></ScrollArea></section>
}

function QueueStatusBadge({ item }: { item: QueuedImage }) {
  if (item.status === "uploaded") return <Badge variant="secondary"><CheckCircle2 className="mr-1 h-3 w-3" />Uploaded</Badge>
  if (item.status === "duplicate") return <Badge variant="outline"><CopyCheck className="mr-1 h-3 w-3" />Duplicate</Badge>
  if (item.status === "failed") return <Badge variant="destructive"><CircleAlert className="mr-1 h-3 w-3" />Needs attention</Badge>
  return <Badge variant="outline">{item.status === "checking" ? "Checking…" : item.status === "uploading" ? "Uploading…" : "Ready"}</Badge>
}

function AssetCard({ asset, editable, onInsert, onEdit, onReplace, onRemove }: { asset: GuidelineAsset; editable: boolean; onInsert: (asset: GuidelineAsset) => void; onEdit: () => void; onReplace: () => void; onRemove: () => void }) {
  return <article className="overflow-hidden rounded-lg border"><div className="aspect-video bg-muted"><img className="h-full w-full object-contain" src={asset.url} alt={asset.alternative_text || "Unlabelled guideline asset"} /></div><div className="space-y-2 p-3"><div className="flex flex-wrap gap-2"><Badge variant="outline">{asset.review_status}</Badge><Badge variant={asset.referenced ? "secondary" : "destructive"}>{asset.referenced ? "Used" : "Unused"}</Badge>{asset.clinically_sensitive && <Badge>Clinical review</Badge>}</div><p className="truncate font-medium" title={asset.original_filename}>{asset.original_filename}</p><p className="text-xs text-muted-foreground">{asset.alternative_text || "Missing alternative text"} · {(asset.size_bytes / 1024).toFixed(1)} KB</p><div className="flex flex-wrap gap-2"><Button size="sm" disabled={!editable || !asset.alternative_text.trim()} title={!asset.alternative_text.trim() ? "Add alternative text before inserting this image" : undefined} onClick={() => onInsert(asset)}>Insert</Button>{editable && <><Button size="sm" variant="outline" onClick={onEdit}><Pencil className="mr-1 h-3 w-3" />Metadata</Button><Button size="sm" variant="outline" onClick={onReplace}><RefreshCw className="mr-1 h-3 w-3" />Replace</Button><Button size="sm" variant="ghost" disabled={asset.referenced} title={asset.referenced ? "Remove the Markdown reference first" : "Remove unused asset"} onClick={onRemove}><Trash2 className="h-3 w-3" /></Button></>}</div></div></article>
}

function BrokenReferences({ references }: { references: string[] }) {
  if (!references.length) return <p className="py-8 text-center text-sm text-muted-foreground">No broken image references.</p>
  return <div className="space-y-2 pr-3">{references.map((reference) => <div key={reference} className="rounded-md border border-destructive/40 p-3"><p className="font-mono text-xs text-destructive">guideline-asset://{reference}</p><p className="mt-1 text-xs text-muted-foreground">Remove this reference from Markdown or upload and insert the intended image.</p></div>)}</div>
}

function AssetMetadataFields({ idPrefix, metadata, setMetadata, compact = false }: { idPrefix: string; metadata: GuidelineAssetMetadata; setMetadata: (value: GuidelineAssetMetadata) => void; compact?: boolean }) {
  return <div className="space-y-3"><div><Label htmlFor={`${idPrefix}-alt`}>Alternative text (required)</Label><Input id={`${idPrefix}-alt`} required value={metadata.alternative_text || ""} onChange={(event) => setMetadata({ ...metadata, alternative_text: event.target.value })} /></div><div><Label htmlFor={`${idPrefix}-caption`}>Caption</Label><Input id={`${idPrefix}-caption`} value={metadata.caption || ""} onChange={(event) => setMetadata({ ...metadata, caption: event.target.value })} /></div>{!compact && <><div><Label htmlFor={`${idPrefix}-source`}>Source</Label><Input id={`${idPrefix}-source`} value={metadata.source || ""} onChange={(event) => setMetadata({ ...metadata, source: event.target.value })} /></div><div className="grid gap-3 sm:grid-cols-2"><div><Label htmlFor={`${idPrefix}-attribution`}>Attribution</Label><Input id={`${idPrefix}-attribution`} value={metadata.attribution || ""} onChange={(event) => setMetadata({ ...metadata, attribution: event.target.value })} /></div><div><Label htmlFor={`${idPrefix}-license`}>License or copyright</Label><Input id={`${idPrefix}-license`} value={metadata.license || ""} onChange={(event) => setMetadata({ ...metadata, license: event.target.value })} /></div></div><div><Label htmlFor={`${idPrefix}-number`}>Figure number</Label><Input id={`${idPrefix}-number`} type="number" min={1} value={metadata.figure_number || ""} onChange={(event) => setMetadata({ ...metadata, figure_number: event.target.value ? Number(event.target.value) : null })} /></div></>}<label className="flex gap-2 text-sm"><input type="checkbox" checked={metadata.clinically_sensitive || false} onChange={(event) => setMetadata({ ...metadata, clinically_sensitive: event.target.checked })} />Clinically sensitive figure</label></div>
}

function metadataFrom(asset: GuidelineAsset): GuidelineAssetMetadata {
  return { alternative_text: asset.alternative_text, caption: asset.caption, source: asset.source, attribution: asset.attribution, license: asset.license, figure_number: asset.figure_number, clinically_sensitive: asset.clinically_sensitive }
}
