import { FieldOption } from "@/types/data-table"
import { riskLevelOptions } from "@/components/clinical-tools/risk-level"
import { toolCategoryService } from "@/services/tool-category.service"

/**
 * Available fields for decision tools table advanced filtering.
 * Only fields the /api/v2/calculators list endpoint can filter on are offered;
 * calculatorService.listTable maps them to the backend's type, status,
 * category_id and risk_level params.
 */
export const decisionToolsAvailableFields: FieldOption[] = [
  {
    label: "Type",
    value: "type",
    type: "select",
    options: [
      { label: "Calculator", value: "calculator" },
      { label: "Decision Tool", value: "decision_tool" },
      { label: "Checklist", value: "checklist" },
    ],
  },
  {
    label: "Status",
    value: "status",
    type: "select",
    options: [
      { label: "Active", value: "active" },
      { label: "Draft", value: "draft" },
      { label: "Archived", value: "archived" },
    ],
  },
  {
    label: "Category",
    value: "category_id",
    type: "select",
    relation: {
      key: "tool-categories",
      loadOptions: async (search) => {
        const term = search.trim().toLowerCase()
        const categories = await toolCategoryService.all()
        return categories.filter((category) => category.name.toLowerCase().includes(term))
      },
      labelField: "name",
      valueField: "id",
    },
  },
  {
    label: "Risk level",
    value: "risk_level",
    type: "select",
    options: riskLevelOptions.map(({ label, value }) => ({ label, value })),
  },
]
