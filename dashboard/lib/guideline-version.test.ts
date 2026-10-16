import { expect, it } from "vitest"
import { suggestNextGuidelineVersion } from "./guideline-version"

it.each([
  ["1.0", [], "1.1"],
  ["2.3", [], "2.4"],
  ["1", [], "2"],
  ["1.0.9", [], "1.0.10"],
  ["1.0", ["1.1", "1.4", "2.9", "1.0.7"], "1.5"],
  ["1.0", [" 1.01 "], "1.2"],
  ["1.009", [], "1.010"],
  ["1.9007199254740992", [], "1.9007199254740993"],
  ["2026.1", ["2026.2"], "2026.3"],
  ["v1.0", [], ""],
  ["2026-final", [], ""],
  ["", [], ""],
  [undefined, [], ""],
  ["9".repeat(100), [], ""],
] as [string | undefined, string[], string][])("suggests the next available label for %s", (source, existing, expected) => {
  expect(suggestNextGuidelineVersion(source, existing)).toBe(expected)
})
