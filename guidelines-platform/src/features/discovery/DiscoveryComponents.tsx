import { Link } from "react-router-dom";

import {
  type PublicDiseasePage,
  type PublicHub,
  type PublicResource,
} from "../../api/public-guidelines";
import { TaxonomyIcon } from "../../components/common/TaxonomyIcon";
import { dateLabel } from "./discovery-utils";
import {
  confirmExternalResource,
  isExternalResourceRoute,
} from "./resource-navigation";

export function LoadingCards() {
  return (
    <div className="discovery-grid" aria-label="Loading">
      <i className="discovery-shimmer" />
      <i className="discovery-shimmer" />
      <i className="discovery-shimmer" />
    </div>
  );
}

export function StateMessage({
  title,
  retry,
}: {
  title: string;
  retry?: () => void;
}) {
  return (
    <div className="discovery-state">
      <h2>{title}</h2>
      {retry && (
        <button className="button button-primary" onClick={retry}>
          Try again
        </button>
      )}
    </div>
  );
}

export function OfflineNotice() {
  return (
    <p className="offline-notice" role="status">
      You are offline. Showing the most recently saved public content.
    </p>
  );
}

export function DiseaseHierarchy({
  diseases,
}: {
  diseases: PublicDiseasePage["items"];
}) {
  const ids = new Set(diseases.map((disease) => disease.id));
  const roots = diseases.filter(
    (disease) => !disease.parent_id || !ids.has(disease.parent_id),
  );
  const children = (id: string) =>
    diseases.filter((disease) => disease.parent_id === id);
  const card = (disease: PublicDiseasePage["items"][number]) => (
    <div key={disease.id}>
      <Link className="discovery-card" to={`/diseases/${disease.slug}`}>
        <TaxonomyIcon icon={disease.icon} label={disease.name} />
        <div>
          <h2>{disease.name}</h2>
          {disease.short_name && <b>{disease.short_name}</b>}
          <p>
            {disease.description ||
              "Open related hubs and approved public resources."}
          </p>
        </div>
      </Link>
      {children(disease.id).length > 0 && (
        <div className="disease-children">{children(disease.id).map(card)}</div>
      )}
    </div>
  );
  return (
    <div className="discovery-grid disease-hierarchy">{roots.map(card)}</div>
  );
}

export function HubCard({ hub }: { hub: PublicHub }) {
  return (
    <Link className="discovery-card" to={`/hubs/${hub.slug}`}>
      <TaxonomyIcon icon={hub.icon} label={hub.name} />
      <div>
        <h2>{hub.name}</h2>
        <p>{hub.description}</p>
      </div>
    </Link>
  );
}

export function ResourceGrid({ resources }: { resources: PublicResource[] }) {
  return resources.length ? (
    <div className="discovery-grid">
      {resources.map((resource) => (
        <ResourceCard
          resource={resource}
          key={`${resource.content_type}:${resource.id}`}
        />
      ))}
    </div>
  ) : (
    <StateMessage title="No eligible public resources are available here yet." />
  );
}

export function ResourceCard({ resource }: { resource: PublicResource }) {
  const external = isExternalResourceRoute(resource.route);
  const absoluteWebRoute = /^https?:\/\//i.test(resource.route ?? "");
  const metadata = [
    resource.issuing_authority || resource.source_organization,
    resource.version ? `Version ${resource.version}` : "",
    resource.publication_date
      ? `Published ${dateLabel(resource.publication_date)}`
      : "",
    resource.effective_at
      ? `Effective ${dateLabel(resource.effective_at)}`
      : "",
    resource.review_at ? `Review ${dateLabel(resource.review_at)}` : "",
    resource.expires_at ? `Expires ${dateLabel(resource.expires_at)}` : "",
  ].filter(Boolean);
  const body = (
    <>
      <span className="resource-type">
        {resource.content_type.replaceAll("_", " ")}
      </span>
      <h3>{resource.title}</h3>
      <p>{resource.description}</p>
      {metadata.length > 0 && <small>{metadata.join(" · ")}</small>}
      {resource.provenance && <small>Source: {resource.provenance}</small>}
    </>
  );
  if (!resource.route || (absoluteWebRoute && !external)) {
    return <article className="resource-card">{body}</article>;
  }
  if (external) {
    return (
      <a
        className="resource-card"
        href={resource.route}
        target="_blank"
        rel="noreferrer"
        onClick={(event) => {
          if (!confirmExternalResource()) event.preventDefault();
        }}
      >
        {body}
      </a>
    );
  }
  return isLocalDiscoveryRoute(resource.route) ? (
    <Link className="resource-card" to={resource.route}>
      {body}
    </Link>
  ) : (
    <a className="resource-card" href={resource.route}>
      {body}
    </a>
  );
}

function isLocalDiscoveryRoute(route: string) {
  return ["/guidelines/", "/diseases/", "/hubs/", "/search"].some(
    (prefix) => route === prefix || route.startsWith(prefix),
  );
}
