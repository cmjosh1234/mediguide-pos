"use client"

import * as React from "react"
import { DynamicIcon, iconNames, type IconName } from "lucide-react/dynamic"
import { cn } from "@/lib/utils"
import { clinicalIconsByName, isEmojiIcon } from "@/components/ui/clinical-icons"

const lucideIconNames = new Set<string>(iconNames)

/**
 * Renders a stored `icon` value, which is either a Lucide icon name
 * (e.g. "heart-pulse") or an emoji. Anything else renders the fallback so
 * raw identifiers never show up as text.
 */
export function StoredIcon({
  icon,
  fallback,
  className,
}: {
  icon?: string
  fallback: React.ReactNode
  className?: string
}) {
  const value = icon?.trim() ?? ""
  const curated = clinicalIconsByName.get(value)
  if (curated) {
    const Icon = curated.icon
    return <Icon className={cn("h-4 w-4 shrink-0", className)} />
  }
  if (lucideIconNames.has(value)) {
    return (
      <DynamicIcon
        name={value as IconName}
        className={cn("h-4 w-4 shrink-0", className)}
        fallback={() => <>{fallback}</>}
      />
    )
  }
  if (isEmojiIcon(value)) {
    return <span className="text-lg leading-none">{value}</span>
  }
  return <>{fallback}</>
}

export function isValidStoredIcon(value: string): boolean {
  const trimmed = value.trim()
  return trimmed === "" || lucideIconNames.has(trimmed) || isEmojiIcon(trimmed)
}
