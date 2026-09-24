# Editorial Review workspace

This page explains how to use **Editorial Review**
(`/admin/guidelines/<guidelineId>/versions/<versionId>/review`) and what
changed in the September 2026 redesign. The rules are unchanged. Every
table, dosage, recommendation, warning, caution, contraindication, procedure,
algorithm, referral criterion and clinically sensitive figure still needs an
individual, audited decision from a reviewer. The redesign only changes where
things are on the screen and how quickly a reviewer can work through the queue.

For the full authoring lifecycle, see
[guideline-authoring-and-publication-workflow.md](guideline-authoring-and-publication-workflow.md).

## Why it changed

On large guidelines, the old page stacked five full-width panels before the
actual workspace:

1. the header;
2. a "blocks require individual review" banner;
3. **Extraction and publication review**, with one red line and a repeated
   "How to resolve" hint for *every* unreviewed block;
4. the bulk-review panel;
5. the completeness report.

A guideline with a few hundred tables therefore opened on several screens of
red text. The source, structure and preview columns sat below all of it.
Pending blocks were listed in four places, and most "errors" were not defects.
They were blocks nobody had reviewed yet. Real defects, such as a broken
section reference or an invalid payload, were lost among them.

## Layout

```
┌──────────────────────────────────────────────────────────────────────┐
│ ←  Editorial review · Version 1.5  [status]   [⚠ 7 blockers ▾] [Validate] [Publish] │  sticky
│ ▓▓▓░░░░░░  38 of 235 high-risk blocks approved      [Review next pending →]        │
├──────────────────────────────────────────────────────────────────────┤
│ [Review] [Bulk approve low-risk] [Completeness report]               │
├──────────────┬───────────────────────────┬───────────────────────────┤
│ Sections     │ Source PDF · p.14 | Reader│ Section: … [⚙ Section]    │
│  2.1 …    3  │                           │ filter · block list       │
│  2.2 …    ✓  │   original PDF, synced to │ ─────────────────────────  │
│  2.3 …    1  │   the selected block's    │ rendered block + metadata │
│  …           │   page                    │ ▸ Correct extraction      │
│              │                           │ [Approve & next] [Reject] ⋯│
└──────────────┴───────────────────────────┴───────────────────────────┘
```

### Header (always visible)

- **Progress bar.** It counts the high-risk and conditional-risk blocks
  approved out of the total that need individual review.
- **Review next pending.** Jumps to the next pending block in document order,
  in any section.
- **Blockers button.** Opens the **Publication checklist** (see below). When
  nothing blocks publication it reads **Ready to publish**.
- **Validate** reruns the publication validator.
- **Publish** stays disabled until validation passes.
- **Regeneration review.** This link back to the Markdown editor appears only
  when you arrived from **Review pending blocks**.

### Publication checklist

Publication errors are grouped by issue code instead of listed one per line:

- **Real defects** are shown first, in red. Examples are duplicate slugs,
  heading-level problems, invalid payloads and broken references. When a
  defect repeats, it is collapsed into one row ("… (+15 more)"). Use **Show N**
  to expand the list, and select any item to go to the affected block or
  section.
- **The review queue** is a single amber row, for example *"179 high-risk
  blocks awaiting publisher review"*, with a **Start reviewing** button. These
  rows are not defects; they mean a reviewer still has work to do.
- **Advisories** (extraction warnings and validation warnings) are folded into
  one collapsed row, because they do not block publication.

Table names in these messages are cleaned up by the backend (see
`guidelineIssueLabel` in `backend/internal/services/guideline_review_service.go`):

- escape sequences left behind by PDF extraction, such as a literal `\n` or
  `\u009f`, are removed;
- whitespace is collapsed;
- labels are cut to 80 characters.

### Review tab

**Sections (left).** This is the document outline. The amber number beside a
section is how many of its blocks are still pending individual review. A green
tick means every high-risk block in that section is approved.

**Source (middle).** The original PDF opens on the page of the selected block.
Switch to **Reader preview** to see the whole section as readers will see it,
in web or mobile width.

**Review block (right).** This column contains:

- the section name. **⚙ Section** opens the section settings dialog (title,
  heading level, slug, move up or down, merge into the previous section);
- a filter: **Pending review only**, **All high-risk blocks** or **All
  blocks**;
- a compact list of the section's blocks. The coloured dot shows the status:
  amber is draft, green is reviewed and red is rejected;
- the selected block, rendered as readers will see it, with its type, status,
  page and extraction confidence;
- **Correct extraction** (collapsed by default), which holds the content type
  selector and the typed JSON payload;
- the action bar: **Approve & next**, **Reject**, and a **⋯** menu with **Start
  new section here** and **Delete block**.

On screens narrower than the `xl` breakpoint, the three columns become
**Sections / Source / Review block** tabs.

### Other tabs

- **Bulk approve low-risk.** The controlled bulk-review queue, with the same
  rules as before (eligible types only, at most 500 blocks, reviewer
  attestation). Selecting a block in the queue opens it in the Review tab with
  the filter set to **All blocks**.
- **Completeness report.** The read-only publication completeness report with
  its JSON and CSV exports. It now loads only when the tab is opened.

## Reviewing a guideline

1. Open **Editorial Review** for the draft version. The workspace opens on the
   **first pending high-risk block**, with the filter set to **Pending review
   only**. (It used to open on the first section, which was usually front
   matter with nothing to review.)
2. Compare the rendered block with the PDF page in the middle column: every
   heading, cell, value, unit, footnote, caption and source.
3. If the block is correct, select **Approve & next** (or press **A**). The
   decision is recorded and the next pending block in document order opens,
   even when it is in another section.
4. If the extraction is wrong, you have two options:
   - correct it at the source: edit the Markdown, save, regenerate, then
     return here; or
   - make an intentional structured correction: open **Correct extraction**,
     edit the payload, and select **Save correction**. This resets the block to
     `draft`, so review it again.

   You cannot approve or reject a block that has an unsaved correction. Save
   it or select **Discard** first.
5. If the block must not be published, select **Reject** (or press **R**). A
   rejected block remains pending until it is corrected and approved.
6. Use the section outline or **Review next pending** to keep going until the
   progress bar is full.
7. If you came from the Markdown editor, select **Regeneration review**, then
   **Refresh approval status** and **Accept regenerated projection**.
8. Open the blockers button, resolve any remaining red items, select
   **Validate**, then **Publish**.

### Keyboard shortcuts

These work in the Review tab when focus is not in a text field, select, menu or
dialog.

| Key | Action |
| --- | --- |
| `J` | Next block (continues into the next pending block when the section is finished) |
| `K` | Previous block in the section |
| `A` | Approve the selected block and move to the next pending one |
| `R` | Reject the selected block |

## Renamed and moved controls

| Before | Now |
| --- | --- |
| Red **Extraction and publication review** card | **N blockers** button → **Publication checklist** |
| "N blocks require individual review" banner | Progress bar and **Review next pending** in the sticky header |
| **Return to regeneration review** | **Regeneration review** (header, only when arriving from the Markdown editor) |
| **Approve block** / **Approve figure** (outline button) | **Approve & next** / **Approve figure & next** (primary button, `A`) |
| **Reject block** / **Reject figure** | **Reject** (`R`) |
| **Artifact** (delete) | **⋯ → Delete block** |
| **Split section here** | **⋯ → Start new section here** |
| Section title / level / slug / move / merge fields above the block list | **⚙ Section** dialog |
| Filter: *Individual review only* / *Pending individual review only* / *All blocks* | *All high-risk blocks* / *Pending review only* / *All blocks* |
| **Rendered preview** column | **Reader preview** toggle in the Source column |
| Bulk review panel and completeness report stacked on the page | **Bulk approve low-risk** and **Completeness report** tabs |

## What did not change

- The APIs, permissions (`guideline.review`), audit records and review states
  (`draft`, `reviewed`, `rejected`) are unchanged.
- Which block types need individual review is still defined by the backend's
  `block_review_policy`. Figures and high-risk types can never be bulk
  approved.
- Publication validation, regeneration acceptance and the recovery gate behave
  exactly as documented in the authoring workflow.

## Known gaps

- The header progress counts `high_risk_types` plus `conditional_risk_types`
  from `block_review_policy`. The publication blocker `unreviewed_high_risk_block`
  counts only the types the backend classifies as requiring individual review.
  The two numbers can therefore differ (for example, 235 in the header and 179
  in the checklist). Publication follows the checklist.
- Duplicate-slug messages still identify sections by UUID instead of title.
- Table titles often contain an entire first cell and undecoded PDF bullet
  glyphs. The review label hides this, but it needs a fix in the ai-worker's
  table extraction.

## Code map

| Concern | Location |
| --- | --- |
| Page, layout, queue and shortcuts | `dashboard/app/(dashboard)/guidelines/[id]/versions/[versionId]/review/page.tsx` |
| Issue grouping and checklist popover | `dashboard/components/guidelines/guideline-review-issues.tsx` (+ `.test.tsx`) |
| Bulk queue | `dashboard/components/guidelines/guideline-bulk-review-panel.tsx` |
| Completeness report | `dashboard/components/guidelines/guideline-completeness-report.tsx` |
| Issue messages and label cleaning | `backend/internal/services/guideline_review_service.go` |
