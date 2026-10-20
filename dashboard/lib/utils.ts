import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function snakeToTitleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .split(" ")
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ")
}

/** Turns a snake_case lifecycle status such as "pending_review" into "Pending review". */
export function formatStatus(value?: string | null) {
  const text = (value ?? "").replace(/_/g, " ").trim()
  return text.charAt(0).toUpperCase() + text.slice(1)
}
