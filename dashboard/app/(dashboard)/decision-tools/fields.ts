import { FieldOption } from "@/types/data-table"

/**
 * Available fields for decision tools table advanced filtering.
 * Only fields the /api/v2/calculators list endpoint can filter on are offered;
 * calculatorService.listTable maps them to the backend's type/status params.
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
]
