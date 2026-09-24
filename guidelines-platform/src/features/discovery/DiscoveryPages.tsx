import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";

import {
  getPublicDisease,
  getPublicHub,
  listPublicDiseases,
  searchPublicContent,
  type PublicDisease,
  type PublicDiseasePage,
  type PublicHub,
  type PublicSearchResult,
} from "../../api/public-guidelines";
import { TaxonomyIcon } from "../../components/common/TaxonomyIcon";
import {
  DiseaseHierarchy,
  HubCard,
  LoadingCards,
  OfflineNotice,
  ResourceCard,
  ResourceGrid,
  StateMessage,
} from "./DiscoveryComponents";
import {
  collectResources,
  countItems,
  dateLabel,
  flattenPillars,
  uniqueResources,
} from "./discovery-utils";

type LoadState<T> =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; value: T; offline?: boolean };

export function DiseaseDirectoryPage() {
  const [search, setSearch] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState<PublicDiseasePage>>({
    status: "loading",
  });
  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(
      () =>
        listPublicDiseases(search, controller.signal)
          .then((value) =>
            setState({ status: "ready", value, offline: value.offline }),
          )
          .catch(
            () => !controller.signal.aborted && setState({ status: "error" }),
          ),
      200,
    );
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [search, attempt]);
  return (
    <section className="page-shell discovery-page">
      <span className="eyebrow">Clinical directory</span>
      <h1>Diseases and conditions</h1>
      <p>
        Browse the official disease taxonomy and all eligible public guidance
        linked to it.
      </p>
      <label className="discovery-search">
        <span>Search by official name or alias</span>
        <input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="For example: Ebola, malaria, diabetes"
        />
      </label>
      {state.status === "loading" && <LoadingCards />}
      {state.status === "error" && (
        <StateMessage
          title="The disease directory could not be loaded."
          retry={() => setAttempt((value) => value + 1)}
        />
      )}
      {state.status === "ready" && state.offline && <OfflineNotice />}
      {state.status === "ready" && state.value.items.length === 0 && (
        <StateMessage title="No active diseases with public content match this search." />
      )}
      {state.status === "ready" && (
        <DiseaseHierarchy diseases={state.value.items} />
      )}
    </section>
  );
}

export function DiseaseDetailPage() {
  const { slug = "" } = useParams();
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState<PublicDisease>>({
    status: "loading",
  });
  useEffect(() => {
    const controller = new AbortController();
    getPublicDisease(slug, controller.signal)
      .then((value) =>
        setState({ status: "ready", value, offline: value.offline }),
      )
      .catch(() => !controller.signal.aborted && setState({ status: "error" }));
    return () => controller.abort();
  }, [slug, attempt]);
  if (state.status === "loading")
    return (
      <section className="page-shell discovery-page">
        <LoadingCards />
      </section>
    );
  if (state.status === "error")
    return (
      <section className="page-shell discovery-page">
        <StateMessage
          title="This disease is unavailable or has no eligible public content."
          retry={() => {
            setState({ status: "loading" });
            setAttempt((v) => v + 1);
          }}
        />
      </section>
    );
  const disease = state.value;
  return (
    <section className="page-shell discovery-page">
      {state.offline && <OfflineNotice />}
      <Link to="/diseases">← Disease directory</Link>
      <span className="eyebrow">Disease</span>
      <h1>{disease.name}</h1>
      <p>{disease.description}</p>
      {disease.aliases.length > 0 && (
        <p>
          <strong>Also known as:</strong> {disease.aliases.join(", ")}
        </p>
      )}
      {disease.children.length > 0 && (
        <>
          <h2>Related conditions</h2>
          <div className="chip-row">
            {disease.children.map((child) => (
              <Link to={`/diseases/${child.slug}`} key={child.id}>
                {child.name}
              </Link>
            ))}
          </div>
        </>
      )}
      <h2>Content hubs</h2>
      {disease.hubs.length ? (
        <div className="discovery-grid">
          {disease.hubs.map((hub) => (
            <HubCard hub={hub} key={hub.id} />
          ))}
        </div>
      ) : (
        <p>No dedicated hub is currently published.</p>
      )}
      <h2>Public resources</h2>
      <ResourceGrid resources={disease.resources} />
    </section>
  );
}

export function HubPage() {
  const { slug = "" } = useParams();
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState<PublicHub>>({
    status: "loading",
  });
  useEffect(() => {
    const controller = new AbortController();
    getPublicHub(slug, controller.signal)
      .then((value) =>
        setState({ status: "ready", value, offline: value.offline }),
      )
      .catch(() => !controller.signal.aborted && setState({ status: "error" }));
    return () => controller.abort();
  }, [slug, attempt]);
  if (state.status === "loading")
    return (
      <section className="page-shell discovery-page">
        <LoadingCards />
      </section>
    );
  if (state.status === "error")
    return (
      <section className="page-shell discovery-page">
        <StateMessage
          title="This content hub is unavailable."
          retry={() => {
            setState({ status: "loading" });
            setAttempt((v) => v + 1);
          }}
        />
      </section>
    );
  const hub = state.value;
  const allItems = flattenPillars(hub.pillars ?? []).flatMap(
    (pillar) => pillar.items,
  );
  const featured = allItems
    .filter((item) => item.featured && item.resource)
    .map((item) => item.resource!);
  const updates = allItems
    .filter((item) => item.resource)
    .map((item) => item.resource!)
    .sort(
      (a, b) =>
        Date.parse(b.publication_date ?? "") -
        Date.parse(a.publication_date ?? ""),
    )
    .slice(0, 5);
  const reports = uniqueResources(
    allItems
      .filter((item) => item.resource?.content_type === "situation_report")
      .map((item) => item.resource!),
  );
  return (
    <section className="page-shell discovery-page hub-page">
      {state.offline && <OfflineNotice />}
      <span className="eyebrow">Clinical content hub</span>
      <h1>{hub.name}</h1>
      <p>{hub.description}</p>
      {hub.diseases.length > 0 && (
        <div className="chip-row" aria-label="Associated diseases">
          {hub.diseases.map((disease) => (
            <Link to={`/diseases/${disease.slug}`} key={disease.id}>
              {disease.name}
            </Link>
          ))}
        </div>
      )}
      {hub.outbreak && (
        <article className="outbreak-banner">
          <span>Active outbreak · {hub.outbreak.status}</span>
          <h2>{hub.outbreak.title}</h2>
          <p>{hub.outbreak.geographic_area}</p>
          <small>Verified {dateLabel(hub.outbreak.last_verified_at)}</small>
          <div className="metrics-grid">
            {(hub.outbreak.metrics ?? []).map((metric, index) => (
              <div key={index}>
                {Object.entries(metric).map(([key, value]) => (
                  <span key={key}>
                    <b>{String(value)}</b> {key.replaceAll("_", " ")}
                  </span>
                ))}
              </div>
            ))}
          </div>
        </article>
      )}
      <h2>Quick access</h2>
      {hub.pillars?.length ? (
        <div className="pillar-grid">
          {hub.pillars.map((pillar) => (
            <Link
              className="pillar-card"
              to={`/hubs/${hub.slug}/pillars/${pillar.slug}`}
              key={pillar.id}
            >
              <TaxonomyIcon
                icon={pillar.icon}
                label={pillar.name}
                className="resource-icon pillar-icon"
              />
              <h3>{pillar.name}</h3>
              <p>{countItems(pillar)} resources</p>
            </Link>
          ))}
        </div>
      ) : (
        <StateMessage title="This hub has no published sections yet." />
      )}
      {featured.length > 0 && (
        <>
          <h2>Featured</h2>
          <ResourceGrid resources={featured} />
        </>
      )}
      {updates.length > 0 && (
        <>
          <h2>Latest updates</h2>
          <ResourceGrid resources={updates} />
        </>
      )}
      {reports.length > 0 && (
        <>
          <h2>Situation reports</h2>
          <ResourceGrid resources={reports} />
        </>
      )}
    </section>
  );
}

export function PillarPage() {
  const { slug = "", pillarSlug = "" } = useParams();
  const [search, setSearch] = useState("");
  const [kind, setKind] = useState("");
  const [state, setState] = useState<LoadState<PublicHub>>({
    status: "loading",
  });
  useEffect(() => {
    const controller = new AbortController();
    getPublicHub(slug, controller.signal)
      .then((value) =>
        setState({ status: "ready", value, offline: value.offline }),
      )
      .catch(() => !controller.signal.aborted && setState({ status: "error" }));
    return () => controller.abort();
  }, [slug]);
  if (state.status === "loading")
    return (
      <section className="page-shell discovery-page">
        <LoadingCards />
      </section>
    );
  if (state.status === "error")
    return (
      <section className="page-shell discovery-page">
        <StateMessage title="This hub section could not be loaded." />
      </section>
    );
  const pillar = flattenPillars(state.value.pillars ?? []).find(
    (item) => item.slug === pillarSlug,
  );
  if (!pillar)
    return (
      <section className="page-shell discovery-page">
        <StateMessage title="This hub section is unavailable." />
      </section>
    );
  const resources = collectResources(pillar).filter(
    (resource) =>
      (!kind || resource.content_type === kind) &&
      `${resource.title} ${resource.description ?? ""}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  const kinds = [
    ...new Set(
      collectResources(pillar).map((resource) => resource.content_type),
    ),
  ];
  return (
    <section className="page-shell discovery-page">
      {state.offline && <OfflineNotice />}
      <Link to={`/hubs/${state.value.slug}`}>← {state.value.name}</Link>
      <span className="eyebrow">Hub section</span>
      <h1>{pillar.name}</h1>
      <p>{pillar.description}</p>
      {pillar.children.length > 0 && (
        <div className="chip-row">
          {pillar.children.map((child) => (
            <Link to={`/hubs/${slug}/pillars/${child.slug}`} key={child.id}>
              {child.name} · {countItems(child)}
            </Link>
          ))}
        </div>
      )}
      <div className="inline-filters">
        <input
          type="search"
          placeholder="Search this section"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select value={kind} onChange={(e) => setKind(e.target.value)}>
          <option value="">All content types</option>
          {kinds.map((value) => (
            <option key={value} value={value}>
              {value.replaceAll("_", " ")}
            </option>
          ))}
        </select>
      </div>
      <p role="status">
        {resources.length} public resource{resources.length === 1 ? "" : "s"}
      </p>
      <ResourceGrid resources={resources} />
    </section>
  );
}

export function SearchPage() {
  const [params, setParams] = useSearchParams();
  const query = params.get("q") ?? "";
  const [draft, setDraft] = useState(query);
  const [type, setType] = useState(params.get("type") ?? "");
  const [disease, setDisease] = useState(params.get("disease") ?? "");
  const [hub, setHub] = useState(params.get("hub") ?? "");
  const [pillar, setPillar] = useState(params.get("pillar") ?? "");
  const [category, setCategory] = useState(params.get("category") ?? "");
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<LoadState<PublicSearchResult[]>>({
    status: "ready",
    value: [],
  });
  const submit = () => {
    if (draft.trim().length < 2) {
      setState({ status: "ready", value: [] });
    } else {
      setState({ status: "loading" });
    }
    setParams({
      q: draft,
      ...(type ? { type } : {}),
      ...(disease ? { disease } : {}),
      ...(hub ? { hub } : {}),
      ...(pillar ? { pillar } : {}),
      ...(category ? { category } : {}),
    });
  };
  useEffect(() => {
    if (query.trim().length < 2) return;
    const controller = new AbortController();
    searchPublicContent(
      query,
      {
        contentType: type || undefined,
        diseaseSlug: disease || undefined,
        hubSlug: hub || undefined,
        pillarSlug: pillar || undefined,
        categoryId: category || undefined,
      },
      controller.signal,
    )
      .then((value) => setState({ status: "ready", value }))
      .catch(() => !controller.signal.aborted && setState({ status: "error" }));
    return () => controller.abort();
  }, [query, type, disease, hub, pillar, category, attempt]);
  return (
    <section className="page-shell discovery-page">
      <span className="eyebrow">Smart search</span>
      <h1>Search all public clinical content</h1>
      <form
        className="search-filter-panel"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <input
          aria-label="Search"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Disease, guideline, hub, algorithm…"
        />
        <select
          aria-label="Content type"
          value={type}
          onChange={(e) => setType(e.target.value)}
        >
          <option value="">All types</option>
          {[
            "disease",
            "hub",
            "pillar",
            "guideline",
            "outbreak",
            "outbreak_document",
            "situation_report",
            "algorithm",
            "clinical_tool",
            "form",
            "drug_reference",
            "approved_external_url",
          ].map((value) => (
            <option value={value} key={value}>
              {value.replaceAll("_", " ")}
            </option>
          ))}
        </select>
        <input
          aria-label="Disease"
          value={disease}
          onChange={(e) => setDisease(e.target.value)}
          placeholder="Disease name, alias or slug"
        />
        <input
          aria-label="Hub"
          value={hub}
          onChange={(e) => setHub(e.target.value)}
          placeholder="Hub slug"
        />
        <input
          aria-label="Pillar"
          value={pillar}
          onChange={(e) => setPillar(e.target.value)}
          placeholder="Pillar slug"
        />
        <input
          aria-label="Category"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          placeholder="Category ID"
        />
        <button className="button button-primary">Search</button>
      </form>
      {state.status === "loading" && <LoadingCards />}
      {state.status === "error" && (
        <StateMessage
          title="Search is temporarily unavailable."
          retry={() => {
            setState({ status: "loading" });
            setAttempt((value) => value + 1);
          }}
        />
      )}{" "}
      {state.status === "ready" && query && !state.value.length && (
        <StateMessage title="No eligible public content matched your search." />
      )}
      {state.status === "ready" && (
        <div className="discovery-grid">
          {state.value.map((result) => (
            <ResourceCard
              resource={{
                ...result,
                description: result.snippet,
                route: result.route,
              }}
              key={`${result.result_type}:${result.id}`}
            />
          ))}
        </div>
      )}
    </section>
  );
}
