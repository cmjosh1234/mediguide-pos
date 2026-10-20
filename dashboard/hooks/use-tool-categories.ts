import { useQuery } from "@tanstack/react-query"
import {
  TOOL_CATEGORIES_COLLECTION,
  toolCategoryService,
  type ToolCategory,
} from "@/services/tool-category.service"

/**
 * All tool categories, cached and refreshed with the categories table, so
 * labels in tool lists and pickers stay in step with category edits.
 */
export function useToolCategories() {
  const query = useQuery({
    queryKey: ["backend", TOOL_CATEGORIES_COLLECTION, "all"],
    queryFn: () => toolCategoryService.all(),
    staleTime: 5 * 60 * 1000,
  })
  const categories: ToolCategory[] = query.data ?? []
  return {
    categories,
    activeCategories: categories.filter((category) => category.status === "active"),
    byId: new Map(categories.map((category) => [category.id, category])),
    loading: query.isLoading,
    error: query.error,
  }
}
