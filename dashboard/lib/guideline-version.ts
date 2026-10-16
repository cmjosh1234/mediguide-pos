/** Suggest a free numeric label within the withdrawn version's numbering series. */
export function suggestNextGuidelineVersion(source: string | undefined, existing: readonly string[]): string {
  const label = source?.trim() || ""
  if (!/^\d+(?:\.\d+)*$/.test(label)) return ""
  const parts = label.split(".")
  const last = parts.pop()!
  const prefix = parts.length ? `${parts.join(".")}.` : ""
  let highest = BigInt(last)
  for (const version of existing) {
    const value = version.trim()
    if (!value.startsWith(prefix)) continue
    const suffix = value.slice(prefix.length)
    if (/^\d+$/.test(suffix) && BigInt(suffix) > highest) highest = BigInt(suffix)
  }
  const next = `${prefix}${(highest + BigInt(1)).toString().padStart(last.length, "0")}`
  return next.length <= 100 ? next : ""
}
