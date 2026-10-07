import { ExternalLink } from "lucide-react";
import { Fragment } from "react";
import { Link } from "react-router-dom";

import {
  type PublicDiseasePage,
  type PublicHub,
  type PublicResource,
} from "../../api/public-guidelines";
import { TaxonomyIcon } from "../../components/common/TaxonomyIcon";
import { dateLabel } from "./discovery-utils";
import { resourceKind } from "./resource-kinds";
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
  // Sub-conditions are listed inside their parent's card rather than as nested
  // cards, so every cell in the grid keeps the same shape.
  const descendants = (id: string): PublicDiseasePage["items"] =>
    diseases
      .filter((disease) => disease.parent_id === id)
      .flatMap((child) => [child, ...descendants(child.id)]);
  return (
    <div className="discovery-grid disease-hierarchy">
      {roots.map((disease) => {
        const includes = descendants(disease.id);
        return (
          <article className="discovery-card disease-card" key={disease.id}>
            <div className="disease-card-head">
              <TaxonomyIcon icon={disease.icon} label={disease.name} />
              <h2>
                <Link
                  className="disease-card-link"
                  to={`/diseases/${disease.slug}`}
                >
                  {disease.name}
                </Link>
              </h2>
              {disease.short_name && (
                <span className="disease-card-abbr">{disease.short_name}</span>
              )}
            </div>
            <p>
              {disease.description ||
                "Open related hubs and approved public resources."}
            </p>
            {includes.length > 0 && (
              <div className="disease-card-includes">
                Includes{" "}
                {includes.map((child, index) => (
                  <Fragment key={child.id}>
                    {index > 0 && ", "}
                    <Link to={`/diseases/${child.slug}`}>{child.name}</Link>
                  </Fragment>
                ))}
              </div>
            )}
          </article>
        );
      })}
    </div>
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
  const route = resource.route ?? "";
  const external = isExternalResourceRoute(route);
  const absoluteWebRoute = /^https?:\/\//i.test(route);
  const linked = route !== "" && !(absoluteWebRoute && !external);
  const { label, Icon, kind, action } = resourceKind(resource.content_type);
  const issuer = resource.issuing_authority || resource.source_organization;
  const dates = (
    [
      ["Published", resource.publication_date],
      ["Effective", resource.effective_at],
      ["Next review", resource.review_at],
      ["Expires", resource.expires_at],
    ] as const
  ).filter(([, value]) => value);
  const card = { className: "resource-card", "data-kind": kind };
  const body = (
    <>
      <div className="resource-card-head">
        <span className="resource-kind">
          <span className="resource-kind-icon" aria-hidden="true">
            <Icon size={18} strokeWidth={1.9} />
          </span>
          {label}
        </span>
        {resource.version && (
          <span className="resource-card-version">
            {/^\d/.test(resource.version)
              ? `v${resource.version}`
              : resource.version}
          </span>
        )}
      </div>
      <h3>{resource.title}</h3>
      {resource.description && <p>{resource.description}</p>}
      {(issuer || resource.provenance || dates.length > 0) && (
        <div className="resource-card-meta">
          {issuer && <strong>{issuer}</strong>}
          {resource.provenance && <span>{resource.provenance}</span>}
          {dates.length > 0 && (
            <dl>
              {dates.map(([term, value]) => (
                <div key={term}>
                  <dt>{term}</dt>
                  <dd>{dateLabel(value)}</dd>
                </div>
              ))}
            </dl>
          )}
        </div>
      )}
      {linked && (
        <span className="resource-card-action">
          {action}
          {external && <ExternalLink aria-hidden="true" size={14} />}
        </span>
      )}
    </>
  );
  if (!linked) {
    return <article {...card}>{body}</article>;
  }
  if (external) {
    return (
      <a
        {...card}
        href={route}
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
  return isLocalDiscoveryRoute(route) ? (
    <Link {...card} to={route}>
      {body}
    </Link>
  ) : (
    <a {...card} href={route}>
      {body}
    </a>
  );
}

function isLocalDiscoveryRoute(route: string) {
  return ["/guidelines/", "/diseases/", "/hubs/", "/situation-reports/", "/search"].some(
    (prefix) => route === prefix || route.startsWith(prefix),
  );
}
