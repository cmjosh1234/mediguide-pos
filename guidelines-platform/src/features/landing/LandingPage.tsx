import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import {
  listPublicGuidelineDocumentKinds,
  listPublicGuidelines,
  resolvePublicAssetUrl,
  type PublicGuideline,
  type PublicGuidelineDocumentKind,
} from "../../api/public-guidelines";
import {
  ArrowIcon,
  BookIcon,
  DownloadIcon,
  ShieldIcon,
} from "../../components/common/Icons";
import { dashboardLoginUrl } from "../../config";

type LibraryState =
  | { status: "loading" }
  | {
      status: "ready";
      items: PublicGuideline[];
      total: number;
      page: number;
      totalPages: number;
    }
  | { status: "error" };

const publicationsPerPage = 6;

export function LandingPage() {
  const [search, setSearch] = useState("");
  const [programArea, setProgramArea] = useState("");
  const [documentKind, setDocumentKind] = useState("");
  const [documentKinds, setDocumentKinds] = useState<
    PublicGuidelineDocumentKind[]
  >([]);
  const [page, setPage] = useState(1);
  const [reloadKey, setReloadKey] = useState(0);
  const [library, setLibrary] = useState<LibraryState>({ status: "loading" });

  useEffect(() => {
    document.title = "MediGuide Clinical Guidelines";
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    listPublicGuidelineDocumentKinds(controller.signal)
      .then(setDocumentKinds)
      .catch(() => {
        if (!controller.signal.aborted && import.meta.env.DEV) {
          console.warn("Public document kinds request failed");
        }
      });
    return () => controller.abort();
  }, [reloadKey]);

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setLibrary({ status: "loading" });
      const startedAt = performance.now();
      listPublicGuidelines(
        {
          search,
          programArea,
          documentKind,
          page,
          perPage: publicationsPerPage,
        },
        controller.signal,
      )
        .then((result) => {
          setLibrary({
            status: "ready",
            items: result.items,
            total: result.total_items,
            page: result.page,
            totalPages: result.total_pages,
          });
          if (import.meta.env.DEV) {
            console.info("Public guideline list loaded", {
              durationMs: Math.round(performance.now() - startedAt),
              resultCount: result.items.length,
            });
          }
        })
        .catch(() => {
          if (!controller.signal.aborted) {
            setLibrary({ status: "error" });
            if (import.meta.env.DEV) {
              console.warn("Public guideline list request failed");
            }
          }
        });
    }, 250);

    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [search, programArea, documentKind, page, reloadKey]);

  const programAreas = useMemo(() => {
    const areas =
      library.status === "ready"
        ? library.items.map((item) => item.program_area).filter(Boolean)
        : [];
    if (programArea) areas.push(programArea);
    return [...new Set(areas)].sort();
  }, [library, programArea]);

  const selectedKindName = documentKind
    ? documentKinds.find((kind) => kind.slug === documentKind)?.name ||
      "Documents"
    : "All publications";

  const selectDocumentKind = (slug: string) => {
    setDocumentKind(slug);
    setPage(1);
  };

  return (
    <>
      <section className="landing-hero">
        <div className="page-shell hero-grid">
          <div className="hero-copy">
            <span className="eyebrow">
              Republic of Uganda · Ministry of Health
            </span>
            <h1>
              Clinical guidance,
              <span> ready when care decisions matter.</span>
            </h1>
            <p>
              Browse published clinical guidelines in a clear, searchable format
              designed for health workers at every level of care.
            </p>
            <div className="hero-actions">
              <a className="button button-primary" href="#guidelines">
                Browse guidelines <ArrowIcon />
              </a>
              <a className="button button-quiet" href={dashboardLoginUrl}>
                Open staff dashboard
              </a>
            </div>
            <div className="trust-line">
              <ShieldIcon />
              <span>Only reviewed and published guidance is shown</span>
            </div>
          </div>

          <div className="hero-publication" aria-hidden="true">
            <div className="hero-book">
              <span>Republic of Uganda</span>
              <div>
                <small>Ministry of Health</small>
                <strong>Clinical Guidelines</strong>
                <em>Published guidance for common health conditions</em>
              </div>
              <b>MEDIGUIDE</b>
            </div>
            <div className="hero-book-shadow" />
          </div>
        </div>
      </section>

      <section className="library-section page-shell" id="guidelines">
        <div className="section-intro">
          <div>
            <span className="eyebrow">Clinical library</span>
            <h2>Available publications</h2>
          </div>
          <p>Select a publication to read its current published version.</p>
        </div>

        {documentKinds.length > 0 && (
          <DocumentKindSections
            items={documentKinds}
            value={documentKind}
            onChange={selectDocumentKind}
          />
        )}

        <div className="library-filters" role="search">
          <label>
            <span>Search publications</span>
            <input
              type="search"
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
                setPage(1);
              }}
              placeholder="Search by title, source, or topic"
            />
          </label>
          <label>
            <span>Program area</span>
            <select
              value={programArea}
              onChange={(event) => {
                setProgramArea(event.target.value);
                setPage(1);
              }}
            >
              <option value="">All program areas</option>
              {programAreas.map((area) => (
                <option value={area} key={area}>
                  {area}
                </option>
              ))}
            </select>
          </label>
          <label>
            <span>Document kind</span>
            <select
              value={documentKind}
              onChange={(event) => selectDocumentKind(event.target.value)}
            >
              <option value="">All document kinds</option>
              {documentKinds.map((kind) => (
                <option value={kind.slug} key={kind.slug}>
                  {kind.name} ({kind.count})
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="visually-hidden" role="status" aria-live="polite">
          {library.status === "ready"
            ? `${library.total} publication${library.total === 1 ? "" : "s"} found`
            : library.status === "loading"
              ? "Loading guidelines"
              : "Guidelines could not be loaded"}
        </div>

        {library.status === "loading" && <GuidelineSkeleton />}

        {library.status === "error" && (
          <div className="library-state">
            <h3>We could not load the guideline library.</h3>
            <p>Check your connection and try again.</p>
            <button
              className="button button-primary"
              onClick={() => setReloadKey((key) => key + 1)}
            >
              Try again
            </button>
          </div>
        )}

        {library.status === "ready" && library.items.length === 0 && (
          <div className="library-state">
            <h3>No published guidelines match these filters.</h3>
            <button
              className="button button-quiet"
              onClick={() => {
                setSearch("");
                setProgramArea("");
                setDocumentKind("");
                setPage(1);
              }}
            >
              Clear filters
            </button>
          </div>
        )}

        {library.status === "ready" && library.items.length > 0 && (
          <section className="document-kind-panel" aria-labelledby="kind-title">
            <header className="document-kind-panel-header">
              <div>
                <span className="eyebrow">Document section</span>
                <h3 id="kind-title">{selectedKindName}</h3>
              </div>
              <span>
                {library.total} document{library.total === 1 ? "" : "s"}
              </span>
            </header>
            <div className="publication-grid">
              {library.items.map((guideline) => (
                <BackendGuidelineCard guideline={guideline} key={guideline.id} />
              ))}
            </div>
            {library.totalPages > 1 && (
              <nav className="library-pagination" aria-label="Publication pages">
                <button
                  type="button"
                  className="button button-quiet"
                  disabled={library.page <= 1}
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                >
                  Previous
                </button>
                <span>
                  Page {library.page} of {library.totalPages}
                </span>
                <button
                  type="button"
                  className="button button-quiet"
                  disabled={library.page >= library.totalPages}
                  onClick={() =>
                    setPage((current) => Math.min(library.totalPages, current + 1))
                  }
                >
                  Next
                </button>
              </nav>
            )}
          </section>
        )}
      </section>

      <section className="about-section" id="about">
        <div className="page-shell about-grid">
          <div>
            <span className="eyebrow">Built for practical use</span>
            <h2>Clinical guidance in a format that is easier to navigate.</h2>
          </div>
          <div className="about-points">
            <article>
              <BookIcon />
              <div>
                <h3>Current published content</h3>
                <p>
                  New versions appear here after review and publication, without
                  rebuilding this website.
                </p>
              </div>
            </article>
            <article>
              <SearchIcon />
              <div>
                <h3>Find guidance quickly</h3>
                <p>
                  Search by publication title, source organization, or clinical
                  topic.
                </p>
              </div>
            </article>
          </div>
        </div>
      </section>
    </>
  );
}

export function DocumentKindSections({
  items,
  value,
  onChange,
}: {
  items: PublicGuidelineDocumentKind[];
  value: string;
  onChange: (slug: string) => void;
}) {
  const total = items.reduce((sum, item) => sum + item.count, 0);
  return (
    <nav className="document-kind-sections" aria-label="Publication sections">
      <button
        type="button"
        className={value === "" ? "active" : undefined}
        aria-pressed={value === ""}
        onClick={() => onChange("")}
      >
        <span>All publications</span>
        <strong>{total}</strong>
      </button>
      {items.map((item) => (
        <button
          type="button"
          className={value === item.slug ? "active" : undefined}
          aria-pressed={value === item.slug}
          onClick={() => onChange(item.slug)}
          key={item.slug}
        >
          <span>{item.name}</span>
          <strong>{item.count}</strong>
        </button>
      ))}
    </nav>
  );
}

export function BackendGuidelineCard({
  guideline,
}: {
  guideline: PublicGuideline;
}) {
  const readerUrl = `/guidelines/${guideline.id}`;
  const downloadUrl = resolvePublicAssetUrl(
    `/api/public/guidelines/${encodeURIComponent(guideline.id)}/original/download`,
  );
  return (
    <article className="publication-card">
      <div className="publication-cover">
        <span>{guideline.country || "Clinical guideline"}</span>
        <strong>{guideline.title.slice(0, 2).toUpperCase()}</strong>
        <small>{guideline.version}</small>
      </div>
      <div className="publication-card-content">
        <span className="publication-publisher">
          {guideline.source_org || guideline.program_area}
        </span>
        <h3>{guideline.title}</h3>
        <p>
          {guideline.description ||
            "Open this publication to read the current clinical guidance."}
        </p>
        <dl className="publication-meta">
          <div>
            <dt>Version</dt>
            <dd>{guideline.version}</dd>
          </div>
          <div>
            <dt>Language</dt>
            <dd>{guideline.language || "en"}</dd>
          </div>
        </dl>
        <div className="card-actions">
          <Link className="card-action card-action-read" to={readerUrl}>
            Read guideline <ArrowIcon />
          </Link>
          {guideline.has_original_document && (
            <a
              className="card-action card-action-download"
              href={downloadUrl}
              download
              aria-label={`Download ${guideline.title}`}
            >
              <DownloadIcon /> Download
            </a>
          )}
        </div>
      </div>
    </article>
  );
}

function GuidelineSkeleton() {
  return (
    <div className="publication-grid" aria-hidden="true">
      {[0, 1].map((item) => (
        <div className="publication-card guideline-skeleton" key={item}>
          <div className="publication-cover" />
          <div className="publication-card-content">
            <i />
            <i />
            <i />
            <i />
          </div>
        </div>
      ))}
    </div>
  );
}

function SearchIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
    >
      <circle cx="11" cy="11" r="6" />
      <path d="m16 16 4 4" />
    </svg>
  );
}
