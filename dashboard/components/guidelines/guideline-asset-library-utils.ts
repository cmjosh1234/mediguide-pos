import type { GuidelineAsset } from "@/services/guideline-assets.service"

export const guidelineImageExtensions = ["png", "jpg", "jpeg", "gif", "webp"] as const
const supportedMimeTypes = new Set(["image/png", "image/jpeg", "image/gif", "image/webp"])

export type GuidelineAssetLibraryFilter =
  | "all"
  | "used"
  | "unused"
  | "pending"
  | "reviewed"
  | "rejected"
  | "broken"

export function validateGuidelineImageFile(file: Pick<File, "name" | "type" | "size">) {
  const extension = file.name.split(".").pop()?.toLowerCase() || ""
  if (extension === "svg" || (!supportedMimeTypes.has(file.type) && !guidelineImageExtensions.includes(extension as typeof guidelineImageExtensions[number]))) {
    return "Use PNG, JPEG, GIF or WebP. SVG files are rejected."
  }
  if (file.size === 0) return "The image file is empty."
  if (file.size > 10 * 1024 * 1024) return "Guideline images may not exceed 10 MB."
  return ""
}

export function assetMatchesLibraryFilter(
  asset: GuidelineAsset,
  query: string,
  filter: GuidelineAssetLibraryFilter,
) {
  const needle = query.trim().toLocaleLowerCase()
  const searchable = [
    asset.original_filename,
    asset.alternative_text,
    asset.caption,
    asset.source,
    asset.attribution,
    asset.license,
  ].join(" ").toLocaleLowerCase()
  if (needle && !searchable.includes(needle)) return false
  switch (filter) {
    case "used": return asset.referenced
    case "unused": return !asset.referenced
    case "pending": return asset.review_status === "draft"
    case "reviewed": return asset.review_status === "reviewed"
    case "rejected": return asset.review_status === "rejected"
    case "broken": return false
    default: return true
  }
}

export async function guidelineFileChecksum(file: Blob) {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer())
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")
}

