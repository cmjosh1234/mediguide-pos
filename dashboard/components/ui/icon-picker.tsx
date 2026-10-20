"use client"

import * as React from "react"
import { ChevronsUpDown, ImageOff, Search, X } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { clinicalIcons, clinicalIconsByName, isEmojiIcon } from "@/components/ui/clinical-icons"
import { StoredIcon } from "@/components/ui/stored-icon"

type IconPickerProps = {
  id?: string
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  invalid?: boolean
}

/**
 * Picks a stored icon value: a curated clinical Lucide icon name or an emoji.
 */
export function IconPicker({ id, value, onChange, disabled, invalid }: IconPickerProps) {
  const [open, setOpen] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const [emoji, setEmoji] = React.useState("")

  const term = search.trim().toLowerCase()
  const visibleIcons = term
    ? clinicalIcons.filter((entry) =>
        `${entry.name} ${entry.label} ${entry.keywords}`.toLowerCase().includes(term),
      )
    : clinicalIcons

  const current = value.trim()
  const currentLabel = current
    ? clinicalIconsByName.get(current)?.label ?? current
    : "No icon"

  const select = (next: string) => {
    onChange(next)
    setOpen(false)
    setSearch("")
    setEmoji("")
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn("w-full justify-between font-normal", invalid && "border-destructive")}
        >
          <span className="flex items-center gap-2">
            <StoredIcon icon={current} fallback={<ImageOff className="h-4 w-4 text-muted-foreground" />} />
            <span className={cn(!current && "text-muted-foreground")}>{currentLabel}</span>
          </span>
          <ChevronsUpDown className="h-4 w-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-3" align="start">
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            aria-label="Search icons"
            className="pl-8"
            placeholder="Search icons…"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
        <div className="mt-3 grid max-h-56 grid-cols-6 gap-1 overflow-y-auto" role="listbox" aria-label="Clinical icons">
          {visibleIcons.map((entry) => {
            const Icon = entry.icon
            const selected = entry.name === current
            return (
              <button
                key={entry.name}
                type="button"
                role="option"
                aria-selected={selected}
                aria-label={entry.label}
                title={entry.label}
                onClick={() => select(entry.name)}
                className={cn(
                  "flex h-10 w-10 items-center justify-center rounded-md border transition-colors",
                  selected ? "border-primary bg-primary/10 text-primary" : "border-transparent hover:bg-accent",
                )}
              >
                <Icon className="h-5 w-5" />
              </button>
            )
          })}
          {visibleIcons.length === 0 ? (
            <p className="col-span-6 py-6 text-center text-sm text-muted-foreground">No icons match.</p>
          ) : null}
        </div>
        <div className="mt-3 flex items-center gap-2 border-t pt-3">
          <Input
            aria-label="Emoji icon"
            placeholder="Or paste an emoji"
            value={emoji}
            onChange={(event) => setEmoji(event.target.value)}
            className="h-8"
          />
          <Button
            type="button"
            size="sm"
            variant="secondary"
            disabled={!isEmojiIcon(emoji.trim())}
            onClick={() => select(emoji.trim())}
          >
            Use
          </Button>
          {current ? (
            <Button type="button" size="sm" variant="ghost" aria-label="Clear icon" onClick={() => select("")}>
              <X className="h-4 w-4" />
            </Button>
          ) : null}
        </div>
      </PopoverContent>
    </Popover>
  )
}
