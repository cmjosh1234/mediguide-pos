import { backendClient } from "@/lib/backend-client";
import type { DomainPageQuery, DomainPageResult } from "@/types/data-table";
import type {
  ModelsToolCategory,
  ServicesToolCategoryInput,
} from "@/types/generated/backend-openapi";

interface Page<T> {
  items: T[];
  page: number;
  per_page: number;
  total_items: number;
  total_pages: number;
}

export type ToolCategoryStatus = "active" | "inactive";

export type ToolCategory = {
  id: string;
  name: string;
  slug: string;
  description: string;
  sort_order: number;
  status: ToolCategoryStatus;
  color: string;
  icon: string;
  tool_count: number;
  created: string;
  updated: string;
};

export type ToolCategoryInput = ServicesToolCategoryInput;

/** Collection key shared by the categories table and its mutations. */
export const TOOL_CATEGORIES_COLLECTION = "tool_categories";

function normalize(value: ModelsToolCategory): ToolCategory {
  return {
    id: value.id || "",
    name: value.name || "",
    slug: value.slug || "",
    description: value.description || "",
    sort_order: value.sort_order ?? 0,
    status: value.status === "inactive" ? "inactive" : "active",
    color: value.color || "",
    icon: value.icon || "",
    tool_count: value.tool_count ?? 0,
    created: value.created_at || "",
    updated: value.updated_at || "",
  };
}

function list(query: Record<string, unknown>) {
  return backendClient.send<Page<ModelsToolCategory>>("/api/v2/tool-categories", {
    query: { page: 1, per_page: 100, ...query },
  });
}

export const toolCategoryService = {
  /** Every category, in display order. */
  async all(status?: ToolCategoryStatus): Promise<ToolCategory[]> {
    const result = await list({ status, sort: "sort_order", order: "asc" });
    return result.items.map(normalize);
  },

  async listTable(query: DomainPageQuery): Promise<DomainPageResult<ToolCategory>> {
    const status = query.filters.find(
      (filter) => filter.field === "status" && filter.condition === "equals",
    )?.value;
    const result = await list({
      page: query.page,
      per_page: query.perPage,
      search: query.search || undefined,
      status: status ? String(status) : undefined,
      sort: "sort_order",
      order: "asc",
    });
    return {
      items: result.items.map(normalize),
      page: result.page,
      perPage: result.per_page,
      totalItems: result.total_items,
      totalPages: result.total_pages,
    };
  },

  async create(input: ToolCategoryInput): Promise<ToolCategory> {
    return normalize(
      await backendClient.send<ModelsToolCategory>("/api/v2/tool-categories", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    );
  },

  async update(id: string, input: ToolCategoryInput): Promise<ToolCategory> {
    return normalize(
      await backendClient.send<ModelsToolCategory>(`/api/v2/tool-categories/${id}`, {
        method: "PATCH",
        body: JSON.stringify(input),
      }),
    );
  },

  async delete(id: string): Promise<void> {
    await backendClient.send<void>(`/api/v2/tool-categories/${id}`, {
      method: "DELETE",
    });
  },
};
