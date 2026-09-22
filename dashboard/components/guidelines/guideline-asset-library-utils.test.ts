import { describe, expect, it } from "vitest"

import type { GuidelineAsset } from "@/services/guideline-assets.service"
import { assetMatchesLibraryFilter, validateGuidelineImageFile } from "./guideline-asset-library-utils"

const asset = (overrides: Partial<GuidelineAsset> = {}): GuidelineAsset => ({
  id: "asset-1",
  version_id: "version-1",
  type: "figure",
  mime_type: "image/png",
  checksum: "checksum",
  size_bytes: 1024,
  original_filename: "treatment-pathway.png",
  alternative_text: "Severe malaria treatment pathway",
  caption: "Treatment pathway",
  source: "Ministry of Health",
  attribution: "Clinical programme",
  license: "Official publication",
  clinically_sensitive: true,
  review_status: "draft",
  reference: "guideline-asset://asset-1",
  referenced: false,
  url: "https://example.test/image.png",
  url_expires_at: "2026-09-22T00:00:00Z",
  created_at: "2026-09-22T00:00:00Z",
  updated_at: "2026-09-22T00:00:00Z",
  ...overrides,
})

describe("guideline asset library utilities", () => {
  it("accepts supported images and rejects SVG, empty and oversized files", () => {
    expect(validateGuidelineImageFile({ name: "figure.png", type: "image/png", size: 100 })).toBe("")
    expect(validateGuidelineImageFile({ name: "figure.webp", type: "", size: 100 })).toBe("")
    expect(validateGuidelineImageFile({ name: "figure.svg", type: "image/svg+xml", size: 100 })).toContain("SVG")
    expect(validateGuidelineImageFile({ name: "figure.png", type: "image/png", size: 0 })).toContain("empty")
    expect(validateGuidelineImageFile({ name: "figure.png", type: "image/png", size: 11 * 1024 * 1024 })).toContain("10 MB")
  })

  it("filters by usage and review status", () => {
    expect(assetMatchesLibraryFilter(asset({ referenced: true }), "", "used")).toBe(true)
    expect(assetMatchesLibraryFilter(asset({ referenced: false }), "", "used")).toBe(false)
    expect(assetMatchesLibraryFilter(asset({ review_status: "reviewed" }), "", "reviewed")).toBe(true)
    expect(assetMatchesLibraryFilter(asset({ review_status: "rejected" }), "", "pending")).toBe(false)
  })

  it("searches filenames and governed metadata case-insensitively", () => {
    expect(assetMatchesLibraryFilter(asset(), "MALARIA", "all")).toBe(true)
    expect(assetMatchesLibraryFilter(asset(), "Ministry", "all")).toBe(true)
    expect(assetMatchesLibraryFilter(asset(), "cholera", "all")).toBe(false)
  })
})
