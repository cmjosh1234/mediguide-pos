"use client"

import * as React from "react"
import { ChevronDown, ChevronUp, Download, UnfoldVertical } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import { DiffLine, DiffWord, lineDiff, markdownHeadings, wordDiff } from "./markdown-authoring"

interface MarkdownDiffViewerProps {
  before: string
  after: string
  beforeLabel: string
  afterLabel?: string
  onDownload: (content: string) => void
}

type DiffEntry = { kind: "line"; line: DiffLine; index: number } | { kind: "gap"; index: number; count: number }
type ChangeRun = { start: number; end: number; firstChange: number; added: boolean; removed: boolean }

export function MarkdownDiffViewer({ before, after, beforeLabel, afterLabel = "Current draft", onDownload }: MarkdownDiffViewerProps) {
  const [sideBySide, setSideBySide] = React.useState(false)
  const [collapseUnchanged, setCollapseUnchanged] = React.useState(true)
  const [expandedGaps, setExpandedGaps] = React.useState<Set<number>>(() => new Set())
  const [changeIndex, setChangeIndex] = React.useState(-1)
  const scrollers = React.useRef<(HTMLDivElement | null)[]>([])
  const syncing = React.useRef(false)
  const diff = React.useMemo(() => lineDiff(before, after), [after, before])
  const changes = React.useMemo(() => diff.map((line, index) => line.type === "same" ? -1 : index).filter((index) => index >= 0), [diff])
  const headings = React.useMemo(() => markdownHeadings(after), [after])
  const lineNumbers = React.useMemo(() => {
    let oldLine = 0
    let newLine = 0
    return diff.map((line) => ({
      old: line.type === "added" ? undefined : ++oldLine,
      new: line.type === "removed" ? undefined : ++newLine,
    }))
  }, [diff])
  const headingIndexesByDiffRow = React.useMemo(() => {
    const headingsByLine = new Map(headings.map((heading, index) => [heading.line, index]))
    const result = new Map<number, number>()
    diff.forEach((line, index) => {
      const afterLine = lineNumbers[index].new
      if (afterLine === undefined) return
      const headingIndex = headingsByLine.get(afterLine)
      if (headingIndex !== undefined) result.set(index, headingIndex)
    })
    return result
  }, [diff, headings, lineNumbers])
  const visible = React.useMemo<DiffEntry[]>(
    () => collapseUnchanged ? collapseLines(diff, expandedGaps) : diff.map((line, index) => ({ kind: "line", line, index })),
    [collapseUnchanged, diff, expandedGaps],
  )
  const runs = React.useMemo(() => changeRuns(diff, changes), [changes, diff])
  const summary = React.useMemo(() => ({
    added: diff.filter((line) => line.type === "added").length,
    removed: diff.filter((line) => line.type === "removed").length,
  }), [diff])
  const activeRow = changeIndex >= 0 ? changes[changeIndex] : -1

  const jumpTo = (next: number) => {
    setChangeIndex(next)
    document.getElementById(`markdown-diff-${changes[next]}`)?.scrollIntoView?.({ block: "center" })
  }
  const navigate = (direction: number) => {
    if (!changes.length) return
    const start = changeIndex < 0 && direction < 0 ? 0 : changeIndex
    jumpTo((start + direction + changes.length) % changes.length)
  }
  const synchronize = (source: HTMLDivElement, target?: HTMLDivElement | null) => {
    if (!target || syncing.current) return
    syncing.current = true
    const available = source.scrollHeight - source.clientHeight
    target.scrollTop = available <= 0 ? 0 : source.scrollTop / available * (target.scrollHeight - target.clientHeight)
    requestAnimationFrame(() => { syncing.current = false })
  }
  const onKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.target instanceof HTMLSelectElement || event.metaKey || event.ctrlKey || event.altKey) return
    if (event.key === "n" || event.key === "j") { event.preventDefault(); navigate(1) }
    if (event.key === "p" || event.key === "k") { event.preventDefault(); navigate(-1) }
  }

  return <div className="space-y-3" onKeyDown={onKeyDown}>
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border bg-muted/30 px-3 py-2 text-xs">
      <div className="flex items-center gap-2">
        <div className="flex overflow-hidden rounded-md border bg-background">
          <Button size="icon" variant="ghost" className="h-8 w-8 rounded-none" aria-label="Previous change" title="Previous change (p)" onClick={() => navigate(-1)} disabled={!changes.length}><ChevronUp className="h-4 w-4" /></Button>
          <Button size="icon" variant="ghost" className="h-8 w-8 rounded-none border-l" aria-label="Next change" title="Next change (n)" onClick={() => navigate(1)} disabled={!changes.length}><ChevronDown className="h-4 w-4" /></Button>
        </div>
        <div className="min-w-28">
          <div className="font-medium tabular-nums">{!changes.length ? "No changes" : changeIndex < 0 ? `${changes.length.toLocaleString()} changes` : `Change ${(changeIndex + 1).toLocaleString()} of ${changes.length.toLocaleString()}`}</div>
          {changes.length > 0 && <div className="mt-1 h-1 overflow-hidden rounded-full bg-muted"><div className="h-full rounded-full bg-primary transition-all" style={{ width: `${Math.max(0, changeIndex + 1) / changes.length * 100}%` }} /></div>}
        </div>
        <span className="rounded bg-green-100 px-1.5 py-0.5 font-mono font-medium text-green-800 dark:bg-green-950 dark:text-green-300">+{summary.added.toLocaleString()}</span>
        <span className="rounded bg-red-100 px-1.5 py-0.5 font-mono font-medium text-red-800 dark:bg-red-950 dark:text-red-300">−{summary.removed.toLocaleString()}</span>
      </div>
      <div className="ml-auto flex flex-wrap items-center gap-4">
        <label className="flex cursor-pointer items-center gap-2"><Switch checked={sideBySide} onCheckedChange={setSideBySide} />Side by side</label>
        <label className="flex cursor-pointer items-center gap-2"><Switch checked={collapseUnchanged} onCheckedChange={(value) => { setCollapseUnchanged(value); setExpandedGaps(new Set()) }} disabled={sideBySide} />Collapse unchanged</label>
        <Button size="sm" variant="outline" className="h-8" onClick={() => onDownload(diff.map((line) => `${line.type === "added" ? "+" : line.type === "removed" ? "-" : " "}${line.text}`).join("\n"))}><Download className="mr-1.5 h-3.5 w-3.5" />Download diff</Button>
      </div>
    </div>
    {headings.length > 0 && !sideBySide && <label className="flex items-center gap-2 text-xs">
      <span className="shrink-0 text-muted-foreground">Go to heading</span>
      <select className="h-8 min-w-0 flex-1 rounded-md border bg-background px-2 text-xs" defaultValue="" onChange={(event) => document.querySelector(`[data-heading-id="markdown-diff-heading-${event.target.value}"]`)?.scrollIntoView?.({ block: "start" })}><option value="">Choose…</option>{headings.map((heading, index) => <option key={`${heading.id}-${index}`} value={index}>{heading.breadcrumb.join(" › ")}</option>)}</select>
    </label>}
    {sideBySide ? <div className="grid grid-cols-2 overflow-hidden rounded-lg border font-mono text-xs">
      {[{ label: beforeLabel, content: before }, { label: afterLabel, content: after }].map((pane, paneIndex) => <div key={pane.label} className={cn("min-w-0", paneIndex === 1 && "border-l")}><div className="border-b bg-muted px-3 py-2 font-sans font-medium">{pane.label}</div><div ref={(node) => { scrollers.current[paneIndex] = node }} onScroll={(event) => synchronize(event.currentTarget, scrollers.current[paneIndex === 0 ? 1 : 0])} className="h-[50vh] overflow-auto py-1">{pane.content.split("\n").map((line, index) => <div key={index} className="flex"><span className="w-12 shrink-0 select-none pr-3 text-right text-muted-foreground/70">{index + 1}</span><span className="min-w-0 flex-1 whitespace-pre-wrap break-words pr-2">{line || " "}</span></div>)}</div></div>)}
    </div> : <div className="flex overflow-hidden rounded-lg border">
      <div tabIndex={0} aria-label="Markdown changes" className="h-[50vh] min-w-0 flex-1 overflow-auto bg-muted/10 py-1 font-mono text-xs focus-visible:outline-none">
        {visible.map((entry) => entry.kind === "line"
          ? <DiffRow key={`${entry.index}-${entry.line.type}`} line={entry.line} counterpart={counterpartOf(diff, entry.index)} index={entry.index} numbers={lineNumbers[entry.index]} active={entry.index === activeRow} headingIndex={headingIndexesByDiffRow.get(entry.index) ?? -1} />
          : <button key={`gap-${entry.index}`} type="button" onClick={() => setExpandedGaps((current) => new Set(current).add(entry.index))} className="my-1 flex w-full items-center justify-center gap-2 border-y border-dashed bg-muted/50 py-1 font-sans text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"><UnfoldVertical className="h-3.5 w-3.5" />Show {entry.count.toLocaleString()} unchanged {entry.count === 1 ? "line" : "lines"}</button>)}
      </div>
      {runs.length > 0 && <div className="relative w-3 shrink-0 border-l bg-muted/40" aria-hidden="true">
        {runs.map((run) => <button key={run.start} type="button" tabIndex={-1} title={`Jump to change ${run.firstChange + 1}`} onClick={() => jumpTo(run.firstChange)} className={cn("absolute inset-x-0.5 min-h-[3px] rounded-sm opacity-80 hover:opacity-100", run.added && run.removed ? "bg-violet-500" : run.added ? "bg-green-500" : "bg-red-500", activeRow >= run.start && activeRow <= run.end && "inset-x-0 opacity-100 ring-2 ring-primary")} style={{ top: `${run.start / diff.length * 100}%`, height: `${(run.end - run.start + 1) / diff.length * 100}%` }} />)}
      </div>}
    </div>}
    {!sideBySide && changes.length > 0 && <p className="text-[11px] text-muted-foreground">Tip: press <kbd className="rounded border bg-muted px-1 font-mono">n</kbd> / <kbd className="rounded border bg-muted px-1 font-mono">p</kbd> to step through changes. The strip on the right maps every change in the document; click it to jump.</p>}
  </div>
}

function DiffRow({ line, counterpart, index, numbers, active, headingIndex }: { line: DiffLine; counterpart?: string; index: number; numbers: { old?: number; new?: number }; active: boolean; headingIndex: number }) {
  const words = counterpart !== undefined ? mergeWords(wordDiff(line.type === "removed" ? line.text : counterpart, line.type === "added" ? line.text : counterpart), line.type) : null
  return <div
    id={`markdown-diff-${index}`}
    data-heading-id={headingIndex >= 0 ? `markdown-diff-heading-${headingIndex}` : undefined}
    className={cn(
      "flex scroll-my-8 border-l-2 border-transparent",
      line.type === "added" && "border-green-500 bg-green-50 text-green-950 dark:bg-green-950/60 dark:text-green-100",
      line.type === "removed" && "border-red-500 bg-red-50 text-red-950 dark:bg-red-950/60 dark:text-red-100",
      active && "outline outline-2 -outline-offset-2 outline-primary",
    )}
  >
    <span className="w-10 shrink-0 select-none pr-2 text-right text-muted-foreground/70">{numbers.old ?? ""}</span>
    <span className="w-10 shrink-0 select-none pr-2 text-right text-muted-foreground/70">{numbers.new ?? ""}</span>
    <span className="w-5 shrink-0 select-none text-center text-muted-foreground">{line.type === "added" ? "+" : line.type === "removed" ? "−" : ""}</span>
    <span className="min-w-0 flex-1 whitespace-pre-wrap break-words pr-3">
      {words
        ? words.map((part, wordIndex) => <span key={wordIndex} className={part.type === "same" ? undefined : line.type === "added" ? "rounded-sm bg-green-300/70 dark:bg-green-700/70" : "rounded-sm bg-red-300/70 dark:bg-red-700/70"}>{part.text}</span>)
        : line.text || " "}
    </span>
  </div>
}

function counterpartOf(diff: DiffLine[], index: number) {
  const line = diff[index]
  if (line.type === "removed" && diff[index + 1]?.type === "added") return diff[index + 1].text
  if (line.type === "added" && diff[index - 1]?.type === "removed") return diff[index - 1].text
  return undefined
}

// Joins consecutive changed tokens (including the whitespace between them) into one highlight.
function mergeWords(words: DiffWord[], side: DiffLine["type"]) {
  const merged: DiffWord[] = []
  for (const part of words) {
    if (part.type !== "same" && part.type !== side) continue
    const previous = merged[merged.length - 1]
    if (previous && previous.type === part.type) previous.text += part.text
    else merged.push({ ...part })
  }
  for (let index = 1; index < merged.length - 1; index += 1) {
    const part = merged[index]
    if (part.type === "same" && !part.text.trim() && merged[index - 1].type !== "same" && merged[index + 1].type !== "same") {
      merged.splice(index - 1, 3, { type: merged[index - 1].type, text: merged[index - 1].text + part.text + merged[index + 1].text })
      index -= 1
    }
  }
  return merged
}

function changeRuns(diff: DiffLine[], changes: number[]) {
  const runs: ChangeRun[] = []
  changes.forEach((index, changePosition) => {
    const previous = runs[runs.length - 1]
    const type = diff[index].type
    if (previous && previous.end === index - 1) {
      previous.end = index
      previous.added ||= type === "added"
      previous.removed ||= type === "removed"
    } else runs.push({ start: index, end: index, firstChange: changePosition, added: type === "added", removed: type === "removed" })
  })
  return runs
}

function collapseLines(diff: DiffLine[], expanded: Set<number>) {
  const context = 3
  const keep = new Set<number>()
  diff.forEach((line, index) => {
    if (line.type === "same") return
    for (let cursor = Math.max(0, index - context); cursor <= Math.min(diff.length - 1, index + context); cursor += 1) keep.add(cursor)
  })
  const result: DiffEntry[] = []
  let index = 0
  while (index < diff.length) {
    if (keep.has(index)) { result.push({ kind: "line", line: diff[index], index }); index += 1; continue }
    const start = index
    while (index < diff.length && !keep.has(index)) index += 1
    if (expanded.has(start)) for (let cursor = start; cursor < index; cursor += 1) result.push({ kind: "line", line: diff[cursor], index: cursor })
    else result.push({ kind: "gap", index: start, count: index - start })
  }
  return result
}
