import { backendClient } from "@/lib/backend-client";
import type {
  ModelsDocumentKind,
  ServicesDocumentKindInput,
} from "@/types/generated/backend-openapi";

export const documentKindsQueryKey = ["document-kinds"] as const;

export type DocumentKindStatus = "active" | "inactive";

/** A kind shared by guideline documents and outbreak documents. */
export interface DocumentKind {
  id: string;
  name: string;
  /** Stable code documents and clients use; it cannot change after creation. */
  slug: string;
  description: string;
  sort_order: number;
  status: DocumentKindStatus;
  /**
   * Documents of this kind (for example forms) are published as the uploaded
   * PDF or Word file, with their text indexed for search but never edited.
   */
  publish_as_uploaded: boolean;
  /** Live guideline documents assigned to this kind. */
  guideline_document_count: number;
  /** Live outbreak documents and resources assigned to this kind. */
  outbreak_document_count: number;
}

export interface DocumentKindInput {
  name: string;
  slug?: string;
  description?: string;
  sort_order?: number;
  status?: DocumentKindStatus;
  publish_as_uploaded?: boolean;
}

interface Page<T> {
  items: T[];
  total_items: number;
}

const path = "/api/v2/document-kinds";

export function normalizeDocumentKind(value: ModelsDocumentKind): DocumentKind {
  return {
    id: value.id || "",
    name: value.name || "",
    slug: value.slug || "",
    description: value.description || "",
    sort_order: value.sort_order ?? 0,
    status: value.status === "inactive" ? "inactive" : "active",
    publish_as_uploaded: value.publish_as_uploaded === true,
    guideline_document_count: value.guideline_document_count ?? 0,
    outbreak_document_count: value.outbreak_document_count ?? 0,
  };
}

/** Options for a kind picker: active kinds, plus the current kind if it was deactivated. */
export function assignableDocumentKinds(
  kinds: DocumentKind[],
  current?: string,
): DocumentKind[] {
  return kinds.filter(
    (kind) =>
      kind.status === "active" ||
      (current !== undefined && (kind.id === current || kind.slug === current)),
  );
}

function payload(input: DocumentKindInput): ServicesDocumentKindInput {
  return {
    name: input.name.trim(),
    // An empty slug asks the backend to derive one from the name on create.
    slug: input.slug?.trim() ?? "",
    description: input.description?.trim() ?? "",
    sort_order: input.sort_order ?? 0,
    status: input.status ?? "active",
    publish_as_uploaded: input.publish_as_uploaded ?? false,
  };
}

export const documentKindService = {
  /** Every kind in display order. Editors also receive inactive kinds. */
  async list(status?: DocumentKindStatus): Promise<DocumentKind[]> {
    const page = await backendClient.send<Page<ModelsDocumentKind>>(path, {
      query: { page: 1, per_page: 100, sort: "sort_order", order: "asc", status },
    });
    return (page.items || []).map(normalizeDocumentKind);
  },
  async create(input: DocumentKindInput) {
    return normalizeDocumentKind(
      await backendClient.send<ModelsDocumentKind>(path, {
        method: "POST",
        body: JSON.stringify(payload(input)),
      }),
    );
  },
  async update(id: string, input: DocumentKindInput) {
    return normalizeDocumentKind(
      await backendClient.send<ModelsDocumentKind>(`${path}/${id}`, {
        method: "PATCH",
        body: JSON.stringify(payload(input)),
      }),
    );
  },
  async delete(id: string) {
    await backendClient.send<void>(`${path}/${id}`, { method: "DELETE" });
  },
};
