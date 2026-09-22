import GithubSlugger from "github-slugger";

export type MarkdownRenderChunk = {
  content: string;
  headingIds: string[];
};

const chapterHeading = /^##\s+\S/;
const safeLargeSectionBoundary = /^#{2,4}\s+\S/;
const targetChunkCharacters = 48_000;

/**
 * Splits large publications at heading boundaries without cutting tables,
 * lists, callouts, or paragraphs in half. Heading ids are assigned with one
 * document-wide slugger so duplicate headings keep the same anchors they have
 * when the Markdown is rendered as a single document.
 */
export function splitMarkdownForProgressiveRendering(
  content: string,
): MarkdownRenderChunk[] {
  const chunks: string[] = [];
  let lines: string[] = [];
  let characters = 0;

  const flush = () => {
    const value = lines.join("\n").trim();
    if (value) chunks.push(value);
    lines = [];
    characters = 0;
  };

  for (const line of content.split("\n")) {
    const startsChapter = chapterHeading.test(line);
    const splitsOversizedSection =
      characters >= targetChunkCharacters && safeLargeSectionBoundary.test(line);
    if (lines.length && (startsChapter || splitsOversizedSection)) flush();
    lines.push(line);
    characters += line.length + 1;
  }
  flush();

  if (chunks.length === 0) return [];
  const slugger = new GithubSlugger();
  return chunks.map((value) => ({
    content: value,
    headingIds: value.split("\n").flatMap((line) => {
      const match = /^(#{1,6})\s+(.+?)\s*#*$/.exec(line);
      return match ? [slugger.slug(stripHeadingMarkdown(match[2]))] : [];
    }),
  }));
}

function stripHeadingMarkdown(value: string) {
  return value
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[*_`~]/g, "")
    .trim();
}
