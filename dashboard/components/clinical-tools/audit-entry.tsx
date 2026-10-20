import { AuditEntry, auditDetails } from "@/app/(dashboard)/outbreaks/audit-history"
import type { ClinicalToolAuditEntry } from "@/services/clinical-tool.service"
import type { OutbreakAuditRecord } from "@/services/outbreaks.service"

const AUDIT_LABELS: Record<string, string> = {
  created: "Draft created",
  updated: "Draft updated",
  validated: "Validated",
  tests_run: "Tests run",
  submitted: "Submitted for review",
  review_commented: "Review comment",
  approved: "Approved",
  published: "Published",
  superseded: "Superseded by a newer version",
  superseded_for_legacy_rollback: "Superseded when the legacy runtime was restored",
  withdrawn: "Withdrawn",
  deleted: "Draft deleted",
  selected: "Made the current version",
  synthetic_development_reopened: "Reopened for development",
}

// Shapes a tool audit entry like an outbreak one, so both trails share AuditEntry.
function asAuditRecord(item: ClinicalToolAuditEntry): OutbreakAuditRecord {
  const statusChanged = item.from_status && item.to_status && item.from_status !== item.to_status
  return {
    id: item.id,
    actor_id: item.actor_id ?? "",
    actor_name: item.actor_name,
    actor_email: item.actor_email,
    action: item.action,
    entity_type: "calculator_version",
    entity_id: "",
    metadata: { ...item.metadata, ...(statusChanged ? { from_status: item.from_status, to_status: item.to_status } : {}) },
    created_at: item.created_at,
  }
}

function auditLabel(action: string) {
  const verb = action.replace(/^calculator\.(version\.)?/, "")
  return AUDIT_LABELS[verb] ?? verb.replaceAll("_", " ").replaceAll(".", " ")
}

/** A tool version audit event, shown like outbreak audit entries: what, when and by whom. */
export function ClinicalToolAuditEntryItem({ item }: { item: ClinicalToolAuditEntry }) {
  const record = asAuditRecord(item)
  return <AuditEntry entry={record} label={auditLabel(item.action)} details={auditDetails(record)} />
}
