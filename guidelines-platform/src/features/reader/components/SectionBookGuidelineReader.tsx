import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";

import {
  getPublicGuidelineChapter,
  resolvePublicAssetUrl,
  type PublicAICitation,
  type PublicGuideline,
  type PublicGuidelineChapter,
  type PublicGuidelineManifest,
  type PublicGuidelineSection,
} from "../../../api/public-guidelines";
import { Brand } from "../../../components/common/Brand";
import { ThemeToggle } from "../../../components/common/ThemeToggle";
import { GuidelineBlockRenderer } from "./GuidelineBlockRenderer";
import { GuidelineAssistant } from "./GuidelineAssistant";
import { bookReaderPublicationGuidance } from "./book-reader-publication-guidance";
import {
  readerBlocks,
  readerSectionTitle,
  readerTitleKey,
} from "./reader-presentation";
import type { SupplementalReaderView } from "./BookGuidelineReader";

type Props = {
  guideline: PublicGuideline;
  manifest: PublicGuidelineManifest;
  sections: PublicGuidelineSection[];
  supplementalViews: SupplementalReaderView[];
  partial: boolean;
  onSelectView: (view: SupplementalReaderView) => void;
  onOpenOriginal: (page?: number) => void;
  onCitation: (citation: PublicAICitation) => void;
};

const initialChapterCount = 3;

export function SectionBookGuidelineReader({
  guideline,
  manifest,
  sections,
  partial,
  onOpenOriginal,
  onCitation,
}: Props) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [fontScale, setFontScale] = useState(1);
  const [activeSectionId, setActiveSectionId] = useState("");
  const [expandedRoots, setExpandedRoots] = useState<Set<string> | null>(null);
  const [chapters, setChapters] = useState<
    Record<string, PublicGuidelineChapter>
  >({});
  const [loading, setLoading] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<Record<string, boolean>>({});
  const [visibleRootCount, setVisibleRootCount] = useState(initialChapterCount);
  const loadMoreSentinel = useRef<HTMLDivElement>(null);
  const inFlight = useRef(
    new Map<string, Promise<PublicGuidelineChapter | undefined>>(),
  );
  const chaptersRef = useRef<Record<string, PublicGuidelineChapter>>({});
  const roots = useMemo(
    () => chapterRoots(guideline, sections),
    [guideline, sections],
  );
  const defaultExpandedRoots = useMemo(
    () =>
      new Set(roots.slice(0, Math.min(2, roots.length)).map((root) => root.id)),
    [roots],
  );
  const visibleExpandedRoots = expandedRoots ?? defaultExpandedRoots;
  const visibleRoots = useMemo(
    () => roots.slice(0, Math.min(visibleRootCount, roots.length)),
    [roots, visibleRootCount],
  );
  const rootLookup = useMemo(
    () => buildRootLookup(roots, sections),
    [roots, sections],
  );
  const navigableSections = useMemo(
    () => sections.filter((section) => rootLookup.has(section.id)),
    [rootLookup, sections],
  );
  const childrenByParent = useMemo(() => {
    const result = new Map<string, PublicGuidelineSection[]>();
    for (const section of navigableSections) {
      if (!section.parent_id) continue;
      const items = result.get(section.parent_id) ?? [];
      items.push(section);
      result.set(section.parent_id, items);
    }
    for (const items of result.values())
      items.sort((a, b) => a.sort_order - b.sort_order);
    return result;
  }, [navigableSections]);
  const filteredSections = useMemo(() => {
    const term = query.trim().toLocaleLowerCase();
    if (!term) return navigableSections;
    return navigableSections.filter((section) =>
      section.title.toLocaleLowerCase().includes(term),
    );
  }, [navigableSections, query]);
  const publicationGuidance = bookReaderPublicationGuidance(
    partial,
    manifest.has_original_pdf,
  );
  const activeRoot = useMemo(
    () =>
      (activeSectionId ? rootLookup.get(activeSectionId) : undefined) ??
      visibleRoots[0] ??
      roots[0],
    [activeSectionId, rootLookup, roots, visibleRoots],
  );
  const activeChapter = activeRoot ? chapters[activeRoot.id] : undefined;
  const onThisPageSections = useMemo(
    () => (activeChapter ? visibleChapterSections(activeChapter) : []),
    [activeChapter],
  );
  const activeRootIndex = activeRoot
    ? roots.findIndex((root) => root.id === activeRoot.id)
    : -1;
  const activeRootTitle = activeRoot
    ? readerSectionTitle(activeRoot.title)
    : undefined;

  const loadChapter = useCallback(
    (rootId: string): Promise<PublicGuidelineChapter | undefined> => {
      if (chaptersRef.current[rootId])
        return Promise.resolve(chaptersRef.current[rootId]);
      const existing = inFlight.current.get(rootId);
      if (existing) return existing;
      setLoading((current) => ({ ...current, [rootId]: true }));
      setErrors((current) => ({ ...current, [rootId]: false }));
      const request = getPublicGuidelineChapter(guideline.id, rootId, manifest)
        .then((chapter) => {
          setChapters((current) => {
            const next = { ...current, [rootId]: chapter };
            chaptersRef.current = next;
            return next;
          });
          return chapter;
        })
        .catch(() => {
          setErrors((current) => ({ ...current, [rootId]: true }));
          return undefined;
        })
        .finally(() => {
          inFlight.current.delete(rootId);
          setLoading((current) => ({ ...current, [rootId]: false }));
        });
      inFlight.current.set(rootId, request);
      return request;
    },
    [guideline.id, manifest],
  );

  useEffect(() => {
    visibleRoots.forEach((root) => {
      void loadChapter(root.id);
    });
  }, [loadChapter, visibleRoots]);

  useEffect(() => {
    const sentinel = loadMoreSentinel.current;
    if (
      !sentinel ||
      visibleRootCount >= roots.length ||
      !("IntersectionObserver" in window)
    )
      return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setVisibleRootCount((count) => Math.min(count + 1, roots.length));
        }
      },
      { rootMargin: "1000px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [roots.length, visibleRootCount]);

  const visitSection = useCallback(
    async (sectionId: string, blockId?: string) => {
      const root = rootLookup.get(sectionId);
      if (!root) return false;
      setExpandedRoots((current) => {
        if ((current ?? defaultExpandedRoots).has(root.id)) return current;
        const next = new Set(current ?? defaultExpandedRoots);
        next.add(root.id);
        return next;
      });
      const rootIndex = roots.findIndex(
        (candidate) => candidate.id === root.id,
      );
      setVisibleRootCount((count) => Math.max(count, rootIndex + 1));
      const chapter = await loadChapter(root.id);
      if (!chapter) return false;
      setSidebarOpen(false);
      const target = blockId ? `block-${blockId}` : `section-${sectionId}`;
      window.history.replaceState(
        null,
        "",
        `${window.location.pathname}${window.location.search}#${encodeURIComponent(target)}`,
      );
      requestAnimationFrame(() =>
        requestAnimationFrame(() =>
          document
            .getElementById(target)
            ?.scrollIntoView({ behavior: "smooth", block: "start" }),
        ),
      );
      setActiveSectionId(sectionId);
      return true;
    },
    [defaultExpandedRoots, loadChapter, rootLookup, roots],
  );

  useEffect(() => {
    const elements = Array.from(
      document.querySelectorAll<HTMLElement>(
        ".section-book-section[id^='section-']",
      ),
    );
    if (!elements.length || !("IntersectionObserver" in window)) return;
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort(
            (left, right) =>
              Math.abs(left.boundingClientRect.top) -
              Math.abs(right.boundingClientRect.top),
          )[0];
        if (!visible?.target.id) return;
        const sectionId = visible.target.id.slice("section-".length);
        setActiveSectionId(sectionId);
        const root = rootLookup.get(sectionId);
        if (root) {
          setExpandedRoots((current) => {
            if ((current ?? defaultExpandedRoots).has(root.id)) return current;
            const next = new Set(current ?? defaultExpandedRoots);
            next.add(root.id);
            return next;
          });
        }
      },
      {
        rootMargin: `-${Math.max(72, window.innerHeight * 0.16)}px 0px -62% 0px`,
        threshold: [0, 0.15, 0.5],
      },
    );
    elements.forEach((element) => observer.observe(element));
    return () => observer.disconnect();
  }, [chapters, defaultExpandedRoots, rootLookup]);

  useEffect(() => {
    if (!activeSectionId) return;
    const item = Array.from(
      document.querySelectorAll<HTMLElement>(
        ".section-book-reader .contents-navigation [data-section-id]",
      ),
    ).find((candidate) => candidate.dataset.sectionId === activeSectionId);
    item?.scrollIntoView({ block: "nearest" });
  }, [activeSectionId]);

  const toggleRoot = (rootId: string) => {
    setExpandedRoots((current) => {
      const next = new Set(current ?? defaultExpandedRoots);
      if (next.has(rootId)) next.delete(rootId);
      else next.add(rootId);
      return next;
    });
  };

  useEffect(() => {
    const visitHash = () => {
      if (!window.location.hash) return;
      let target = window.location.hash.slice(1);
      try {
        target = decodeURIComponent(target);
      } catch {
        /* literal hash */
      }
      if (!target.startsWith("section-")) return;
      void visitSection(target.slice("section-".length));
    };
    const frame = window.requestAnimationFrame(visitHash);
    window.addEventListener("hashchange", visitHash);
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("hashchange", visitHash);
    };
  }, [visitSection]);

  const openCitation = async (citation: PublicAICitation) => {
    setAssistantOpen(false);
    if (citation.section_id) {
      if (await visitSection(citation.section_id, citation.block_id)) return;
    }
    onCitation(citation);
  };

  const printAll = async () => {
    setVisibleRootCount(roots.length);
    await Promise.all(roots.map((root) => loadChapter(root.id)));
    window.setTimeout(() => window.print(), 0);
  };

  return (
    <div
      className="reader-site book-reader section-book-reader"
      style={{ "--reader-font-scale": fontScale } as React.CSSProperties}
    >
      <a className="skip-link" href="#guideline-document">
        Skip to clinical content
      </a>
      <header className="reader-header">
        <button
          className="reader-menu-button icon-button"
          type="button"
          aria-label="Open contents"
          onClick={() => setSidebarOpen(true)}
        >
          ☰
        </button>
        <Link className="reader-title" to="/">
          <span>MG</span>
          <strong>{guideline.title}</strong>
        </Link>
        <div className="reader-actions">
          <Link to="/">Guideline library</Link>
          <button
            className="reader-font-button icon-button"
            type="button"
            aria-label="Change text size"
            onClick={() =>
              setFontScale((value) =>
                value >= 1.2 ? 1 : Number((value + 0.1).toFixed(1)),
              )
            }
          >
            A
          </button>
          <button
            className="reader-ai-button"
            type="button"
            onClick={() => setAssistantOpen(true)}
          >
            ✦ Ask AI
          </button>
          <ThemeToggle />
          <button
            className="icon-button"
            type="button"
            aria-label="Print guideline"
            onClick={() => void printAll()}
          >
            ⎙
          </button>
        </div>
      </header>

      <aside
        className={`reader-sidebar ${sidebarOpen ? "is-open" : ""}`}
        aria-label="Guideline contents"
      >
        <div className="reader-sidebar-brand">
          <Brand />
          <button
            className="sidebar-close icon-button"
            type="button"
            aria-label="Close contents"
            onClick={() => setSidebarOpen(false)}
          >
            ×
          </button>
        </div>
        <div className="sidebar-publication">
          <span>Published guideline</span>
          <strong>{guideline.title}</strong>
          <small>
            {guideline.source_org}
            {guideline.version ? ` · Version ${guideline.version}` : ""}
          </small>
        </div>
        <label className="reader-search">
          <SearchIcon />
          <span className="visually-hidden">Search chapter titles</span>
          <input
            type="search"
            value={query}
            placeholder="Search sections…"
            onChange={(event) => setQuery(event.target.value)}
          />
        </label>
        <nav
          className="contents-navigation"
          aria-label={
            query.trim() ? "Section search results" : "Table of contents"
          }
        >
          <div className="toc-toolbar">
            <span>
              {query.trim()
                ? `${filteredSections.length} result${filteredSections.length === 1 ? "" : "s"}`
                : "Contents"}
            </span>
            {!query.trim() && (
              <small>{navigableSections.length} sections</small>
            )}
          </div>
          {query.trim() ? (
            <div className="section-toc-search-results">
              {filteredSections.map((section) => {
                const root = rootLookup.get(section.id);
                return (
                  <button
                    type="button"
                    key={section.id}
                    data-section-id={section.id}
                    className={activeSectionId === section.id ? "active" : ""}
                    aria-current={
                      activeSectionId === section.id ? "location" : undefined
                    }
                    onClick={() => void visitSection(section.id)}
                  >
                    <strong>{section.title}</strong>
                    {root && root.id !== section.id && (
                      <small>{root.title}</small>
                    )}
                  </button>
                );
              })}
              {!filteredSections.length && (
                <p className="empty-search">
                  No section matches “{query}”. Try a condition, treatment,
                  medicine, or chapter title.
                </p>
              )}
            </div>
          ) : (
            roots.map((root) => {
              const expanded = visibleExpandedRoots.has(root.id);
              const descendants = descendantSections(root.id, childrenByParent);
              const activeInRoot =
                activeSectionId === root.id ||
                descendants.some((section) => section.id === activeSectionId);
              const display = readerSectionTitle(root.title);
              return (
                <section
                  className={`section-toc-group ${activeInRoot ? "is-active" : ""}`}
                  key={root.id}
                >
                  <div className="section-toc-root-row">
                    <button
                      type="button"
                      className={`section-toc-root ${activeSectionId === root.id ? "active" : ""}`}
                      data-section-id={root.id}
                      aria-current={
                        activeSectionId === root.id ? "location" : undefined
                      }
                      onClick={() => void visitSection(root.id)}
                    >
                      {display.label && <small>{display.label}</small>}
                      <strong>{display.title}</strong>
                    </button>
                    {descendants.length > 0 && (
                      <button
                        className="section-toc-toggle"
                        type="button"
                        aria-label={`${expanded ? "Collapse" : "Expand"} ${root.title}`}
                        aria-expanded={expanded}
                        onClick={() => toggleRoot(root.id)}
                      >
                        {expanded ? "−" : "+"}
                      </button>
                    )}
                  </div>
                  {expanded && descendants.length > 0 && (
                    <div className="section-toc-children">
                      {descendants.map((section) => (
                        <button
                          type="button"
                          key={section.id}
                          data-section-id={section.id}
                          className={`toc-link toc-depth-${Math.min(4, Math.max(2, section.level + 1))} ${activeSectionId === section.id ? "active" : ""}`}
                          aria-current={
                            activeSectionId === section.id
                              ? "location"
                              : undefined
                          }
                          onClick={() => void visitSection(section.id)}
                        >
                          {section.title}
                        </button>
                      ))}
                    </div>
                  )}
                </section>
              );
            })
          )}
        </nav>
        <div className="reader-sidebar-footer">
          Ministry of Health · Published clinical guidance
        </div>
      </aside>
      {sidebarOpen && (
        <button
          className="reader-scrim"
          type="button"
          aria-label="Close contents"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      <main className="reader-main" id="guideline-document">
        <div className="section-reader-layout">
          <article className="reader-column">
            <header className="guideline-metadata">
              <span className="eyebrow">
                {guideline.program_area || "Clinical guideline"}
              </span>
              <h1>{guideline.title}</h1>
              {guideline.description && <p>{guideline.description}</p>}
              {/* <dl>
                <Meta label="Source" value={guideline.source_org} />
                <Meta label="Version" value={guideline.version} />
                <Meta
                  label="Published"
                  value={formatDate(guideline.publication_date)}
                />
                <Meta
                  label="Review date"
                  value={formatDate(guideline.review_date)}
                />
                <Meta label="Language" value={guideline.language} />
              </dl> */}
              {/* <div className="book-reader-links">
                {supplementalViews.map((view) => (
                  <button
                    type="button"
                    key={view}
                    onClick={() => onSelectView(view)}
                  >
                    {view === "chapters"
                      ? "Reviewed chapters"
                      : `Reviewed ${view}`}
                  </button>
                ))}
                {publicationGuidance.showOriginal && (
                  <button type="button" onClick={() => onOpenOriginal()}>
                    Original PDF
                  </button>
                )}
              </div> */}
            </header>
            {publicationGuidance.partialNotice && (
              <div className="partial-extraction-notice" role="status">
                {publicationGuidance.partialNotice}
              </div>
            )}
            <div className="section-book-content" aria-live="polite">
              {visibleRoots.map((root) => (
                <Chapter
                  guidelineId={guideline.id}
                  key={root.id}
                  root={root}
                  chapter={chapters[root.id]}
                  loading={loading[root.id]}
                  error={errors[root.id]}
                  onRetry={() => void loadChapter(root.id)}
                  onOpenSourcePage={onOpenOriginal}
                />
              ))}
              {visibleRootCount < roots.length && (
                <div
                  className="progressive-markdown-more"
                  ref={loadMoreSentinel}
                >
                  <button
                    type="button"
                    onClick={() =>
                      setVisibleRootCount((count) =>
                        Math.min(count + 1, roots.length),
                      )
                    }
                  >
                    Load next chapter
                  </button>
                </div>
              )}
            </div>
          </article>
          <aside className="reader-context-toc" aria-label="On this page">
            <div className="reader-context-toc-inner">
              <span className="reader-context-label">On this page</span>
              {activeRoot && (
                <>
                  {activeRootIndex >= 0 && (
                    <small>
                      {activeRootTitle?.label
                        ? `${activeRootTitle.label} · Part ${activeRootIndex + 1} of ${roots.length}`
                        : `Part ${activeRootIndex + 1} of ${roots.length}`}
                    </small>
                  )}
                  <strong>{activeRootTitle?.title ?? activeRoot.title}</strong>
                </>
              )}
              {activeChapter ? (
                <nav>
                  {onThisPageSections.map((section) => (
                    <button
                      key={section.id}
                      type="button"
                      className={activeSectionId === section.id ? "active" : ""}
                      aria-current={
                        activeSectionId === section.id ? "location" : undefined
                      }
                      onClick={() => void visitSection(section.id)}
                    >
                      {section.title}
                    </button>
                  ))}
                </nav>
              ) : (
                activeRoot && <p>Loading chapter sections…</p>
              )}
            </div>
          </aside>
        </div>
      </main>
      {!assistantOpen && (
        <button
          className="assistant-fab"
          type="button"
          aria-label="Ask AI about this guideline"
          onClick={() => setAssistantOpen(true)}
        >
          ✦ <span>Ask AI</span>
        </button>
      )}
      <GuidelineAssistant
        key={guideline.id}
        guideline={guideline}
        open={assistantOpen}
        onClose={() => setAssistantOpen(false)}
        onCitation={(citation) => void openCitation(citation)}
      />
    </div>
  );
}

function Chapter({
  guidelineId,
  root,
  chapter,
  loading,
  error,
  onRetry,
  onOpenSourcePage,
}: {
  guidelineId: string;
  root: PublicGuidelineSection;
  chapter?: PublicGuidelineChapter;
  loading?: boolean;
  error?: boolean;
  onRetry: () => void;
  onOpenSourcePage: (page?: number) => void;
}) {
  if (!chapter)
    return (
      <section className="section-book-chapter" id={`section-${root.id}`}>
        <h2>{root.title}</h2>
        {loading ? (
          <p>Loading chapter…</p>
        ) : error ? (
          <>
            <p>Chapter could not be loaded.</p>
            <button className="button button-outline" onClick={onRetry}>
              Try again
            </button>
          </>
        ) : null}
      </section>
    );
  const bySection = new Map<string, typeof chapter.blocks>();
  chapter.blocks.forEach((block) => {
    if (!block.section_id) return;
    const items = bySection.get(block.section_id) ?? [];
    items.push(block);
    bySection.set(block.section_id, items);
  });
  const visibleSections = visibleChapterSections(chapter);
  return (
    <section className="section-book-chapter">
      {visibleSections.map((section) => {
        const blocks = readerBlocks(section, bySection.get(section.id) ?? []);
        const Heading =
          section.level <= 2 ? "h2" : section.level === 3 ? "h3" : "h4";
        return (
          <section
            className="section-book-section"
            id={`section-${section.id}`}
            key={section.id}
          >
            <header>
              <span className="eyebrow">
                {section.page_start
                  ? `Page ${section.page_start}${section.page_end && section.page_end !== section.page_start ? `–${section.page_end}` : ""}`
                  : ""}
              </span>
              <Heading>{section.title}</Heading>
            </header>
            {blocks.map((block) => {
              const assetID =
                block.type === "figure" &&
                typeof block.content.asset_id === "string"
                  ? block.content.asset_id
                  : undefined;
              const figure = assetID
                ? {
                    id: block.id,
                    section_id: block.section_id,
                    sort_order: block.sort_order,
                    page_start: block.page_start,
                    page_end: block.page_end,
                    content: {
                      type: "figure",
                      asset_id: assetID,
                      caption:
                        typeof block.content.caption === "string"
                          ? block.content.caption
                          : undefined,
                      alternative_text:
                        typeof block.content.alternative_text === "string"
                          ? block.content.alternative_text
                          : "Guideline figure",
                    },
                    asset: {
                      asset_id: assetID,
                      type: "figure",
                      mime_type: "image/*",
                      url: resolvePublicAssetUrl(
                        `/api/public/guidelines/${encodeURIComponent(guidelineId)}/assets/${encodeURIComponent(assetID)}/download`,
                      ),
                      expires_at: "",
                    },
                  }
                : undefined;
              return (
                <GuidelineBlockRenderer
                  key={block.id}
                  block={block}
                  figure={figure}
                  onOpenSourcePage={(page) => onOpenSourcePage(page)}
                />
              );
            })}
          </section>
        );
      })}
    </section>
  );
}

function chapterRoots(
  guideline: PublicGuideline,
  sections: PublicGuidelineSection[],
) {
  const roots = sections
    .filter((section) => !section.parent_id)
    .sort((a, b) => a.sort_order - b.sort_order);
  const wrapper =
    roots.length === 1 &&
    readerTitleKey(roots[0].title) === readerTitleKey(guideline.title)
      ? roots[0]
      : undefined;
  if (wrapper) {
    const children = sections
      .filter((section) => section.parent_id === wrapper.id)
      .sort((a, b) => a.sort_order - b.sort_order);
    return children.length ? children : [wrapper];
  }
  if (roots.length) return roots;
  const minimumLevel = sections.reduce(
    (level, section) => Math.min(level, section.level),
    Number.POSITIVE_INFINITY,
  );
  return sections
    .filter((section) => section.level === minimumLevel)
    .sort((a, b) => a.sort_order - b.sort_order);
}

function buildRootLookup(
  roots: PublicGuidelineSection[],
  sections: PublicGuidelineSection[],
) {
  const byId = new Map(sections.map((section) => [section.id, section]));
  const rootIDs = new Set(roots.map((root) => root.id));
  const rootByID = new Map(roots.map((root) => [root.id, root]));
  const result = new Map<string, PublicGuidelineSection>();
  for (const section of sections) {
    let current: PublicGuidelineSection | undefined = section;
    while (current) {
      if (rootIDs.has(current.id)) {
        result.set(section.id, rootByID.get(current.id)!);
        break;
      }
      current = current.parent_id ? byId.get(current.parent_id) : undefined;
    }
  }
  return result;
}

function descendantSections(
  rootId: string,
  childrenByParent: Map<string, PublicGuidelineSection[]>,
) {
  const result: PublicGuidelineSection[] = [];
  const queue = [...(childrenByParent.get(rootId) ?? [])];
  while (queue.length) {
    const section = queue.shift()!;
    result.push(section);
    queue.unshift(...(childrenByParent.get(section.id) ?? []));
  }
  return result;
}

function visibleChapterSections(chapter: PublicGuidelineChapter) {
  const bySection = new Map<string, typeof chapter.blocks>();
  chapter.blocks.forEach((block) => {
    if (!block.section_id) return;
    const items = bySection.get(block.section_id) ?? [];
    items.push(block);
    bySection.set(block.section_id, items);
  });
  const sectionByID = new Map(
    chapter.sections.map((section) => [section.id, section]),
  );
  const visible = new Set<string>([chapter.root_section_id]);
  for (const sectionID of bySection.keys()) {
    let section = sectionByID.get(sectionID);
    while (section) {
      visible.add(section.id);
      section = section.parent_id
        ? sectionByID.get(section.parent_id)
        : undefined;
    }
  }
  return chapter.sections.filter((section) => visible.has(section.id));
}

// function Meta({ label, value }: { label: string; value?: string }) {
//   return value ? (
//     <div>
//       <dt>{label}</dt>
//       <dd>{value}</dd>
//     </div>
//   ) : null;
// }
// function formatDate(value?: string) {
//   if (!value) return "";
//   const parsed = new Date(value);
//   return Number.isNaN(parsed.valueOf())
//     ? value
//     : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
//         parsed,
//       );
// }
function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <circle
        cx="11"
        cy="11"
        r="6.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
      />
      <path
        d="m16 16 4 4"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}
