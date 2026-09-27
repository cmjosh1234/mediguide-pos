import type {
  PublicGuideline,
  PublicGuidelineBlock,
  PublicGuidelineSection,
} from "../../../api/public-guidelines";

export function readerTitleKey(value: string): string {
  return value
    .replace(/[*_`~#]/g, "")
    .normalize("NFKC")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim()
    .toLocaleLowerCase();
}

export type ReaderSectionTitle = {
  label?: string;
  title: string;
  chapterNumber?: string;
};

/**
 * Separates an explicit chapter prefix supplied by the publication from the
 * visible title. We must never derive clinical chapter numbers from array/root
 * position because front matter is commonly stored as top-level sections too.
 *
 * Examples:
 *   "Chapter 1: Epidemiology of Diabetes" -> { label: "Chapter 1", title: "Epidemiology of Diabetes" }
 *   "Chapter 2. Definition and Classification" -> { label: "Chapter 2", title: "Definition and Classification" }
 *   "Executive Summary" -> { title: "Executive Summary" }
 */
export function readerSectionTitle(value: string): ReaderSectionTitle {
  const title = value.trim();
  const match = /^(?:chapter|chap\.?)[\s-]+([0-9]+|[ivxlcdm]+)\s*(?:[:.\-–—]\s*|\s+)(.+)$/iu.exec(title);
  if (!match) return { title };
  const chapterNumber = match[1].toLocaleUpperCase();
  return {
    label: `Chapter ${chapterNumber}`,
    chapterNumber,
    title: match[2].trim(),
  };
}

export function isDocumentTitleWrapper(
  guideline: PublicGuideline,
  section: PublicGuidelineSection,
  sections: PublicGuidelineSection[],
): boolean {
  return (
    section.level === 1 &&
    !section.parent_id &&
    readerTitleKey(section.title) === readerTitleKey(guideline.title) &&
    sections.some((candidate) => candidate.parent_id === section.id)
  );
}

/**
 * Removes the extracted heading block already represented by the section
 * heading. The block remains in the API payload for provenance, search, and
 * RAG; this is presentation-only deduplication.
 */
export function readerBlocks(
  section: PublicGuidelineSection,
  blocks: PublicGuidelineBlock[],
): PublicGuidelineBlock[] {
  const ordered = [...blocks].sort((left, right) => left.sort_order - right.sort_order);
  const first = ordered[0];
  if (
    first?.type === "heading" &&
    readerTitleKey(String(first.content.text ?? "")) === readerTitleKey(section.title)
  ) {
    return ordered.slice(1);
  }
  return ordered;
}

export function readerSections(
  guideline: PublicGuideline,
  sections: PublicGuidelineSection[],
  blocks: PublicGuidelineBlock[],
): PublicGuidelineSection[] {
  return sections.filter((section) => {
    if (!isDocumentTitleWrapper(guideline, section, sections)) return true;
    return readerBlocks(
      section,
      blocks.filter((block) => block.section_id === section.id),
    ).length > 0;
  });
}

export function readerNavigationSections(
  guideline: PublicGuideline,
  sections: PublicGuidelineSection[],
): PublicGuidelineSection[] {
  return sections.filter(
    (section) => !isDocumentTitleWrapper(guideline, section, sections),
  );
}

/** The book reader owns the visible H1, so the Markdown body starts after it. */
export function removeLeadingDocumentTitle(content: string): string {
  const match = /^\s*#\s+.+?\s*#*\r?\n+/.exec(content);
  return match ? content.slice(match[0].length) : content;
}
