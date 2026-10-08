"use client";

import { getBackendClient } from "@/lib/backend-client";
import type {
  ServicesChildContentInput,
  ServicesOutbreakAdminDTO,
  ServicesOutbreakDeleteResult,
  ServicesOutbreakInput,
  ServicesOutbreakMetricsInput,
  ServicesOutbreakNotificationCampaignInput,
  ServicesOutbreakMetric,
  ServicesOutbreakResourceAdminDTO,
  ServicesOutbreakUpdateAdminDTO,
  ServicesSituationReportAdminDTO,
  ServicesSituationReportAttachmentDTO,
  ServicesSituationReportAttachmentInput,
  ServicesSituationReportInput,
  ServicesResourceCorrectionInput,
  ServicesTransitionInput,
  ServicesNotificationCampaignDTO,
  ServicesPublicGuideline,
} from "@/types/generated/backend-openapi";

export type OutbreakRecord = ServicesOutbreakAdminDTO;
export type OutbreakUpdateRecord = ServicesOutbreakUpdateAdminDTO;
export type OutbreakResourceRecord = ServicesOutbreakResourceAdminDTO;
export type SituationReportRecord = ServicesSituationReportAdminDTO;
export type SituationReportAttachmentRecord = ServicesSituationReportAttachmentDTO;
export type SituationReportAttachmentInput = ServicesSituationReportAttachmentInput;
export type OutbreakInput = ServicesOutbreakInput;
export type OutbreakMetricsInput = ServicesOutbreakMetricsInput;
export type SituationReportInput = ServicesSituationReportInput;
export type ChildContentInput = ServicesChildContentInput;
export type OutbreakCampaignInput = ServicesOutbreakNotificationCampaignInput;
export type OutbreakMetric = ServicesOutbreakMetric;
export type OutbreakDeleteResult = ServicesOutbreakDeleteResult;
export type PublishedGuidelineRecord = ServicesPublicGuideline;

export interface PagedResult<T> {
  items: T[];
  page: number;
  per_page: number;
  total_items: number;
  total_pages: number;
}
export interface OutbreakAuditRecord {
  id: string;
  actor_id: string;
  actor_name?: string;
  actor_email?: string;
  /** Display names for the people and records the metadata refers to by ID. */
  labels?: Record<string, string>;
  action: string;
  entity_type: string;
  entity_id: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface OutbreakListQuery {
  page?: number;
  per_page?: number;
  search?: string;
  status?: string;
  disease?: string;
  area?: string;
  region_id?: string;
  visual_tone?: string;
  outbreak_id?: string;
  effective_from?: string;
  effective_to?: string;
  updated_from?: string;
  updated_to?: string;
  sort?: string;
  order?: "asc" | "desc";
}

const client = () => getBackendClient();

function transition(path: string, input: ServicesTransitionInput) {
  return client().send(path, { method: "POST", body: JSON.stringify(input) });
}

export const outbreaksService = {
  list(query: OutbreakListQuery = {}) {
    return client().send<PagedResult<OutbreakRecord>>("/api/v2/outbreaks", {
      query: { ...query },
    });
  },
  listPublishedGuidelines(search = "", documentKind = "") {
    return client().send<PagedResult<PublishedGuidelineRecord>>(
      "/api/public/guidelines",
      { query: { search, document_kind: documentKind, page: 1, per_page: 100 } },
    );
  },
  get(id: string) {
    return client().send<OutbreakRecord>(`/api/v2/outbreaks/${id}`);
  },
  create(input: OutbreakInput) {
    return client().send<OutbreakRecord>("/api/v2/outbreaks", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  update(id: string, input: OutbreakInput) {
    return client().send<OutbreakRecord>(`/api/v2/outbreaks/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    });
  },
  /** Deletes the outbreak with its own updates and resources; linked situation reports and content hubs are kept and unlinked. */
  delete(id: string, lockVersion: number, reason: string) {
    return client().send<OutbreakDeleteResult>(`/api/v2/outbreaks/${id}`, {
      method: "DELETE",
      query: { lock_version: lockVersion },
      body: JSON.stringify({ reason }),
    });
  },
  updateMetrics(id: string, input: OutbreakMetricsInput) {
    return client().send<OutbreakRecord>(`/api/v2/outbreaks/${id}/metrics`, {
      method: "PATCH",
      body: JSON.stringify(input),
    });
  },
  remove(id: string, lockVersion: number) {
    return client().send<void>(`/api/v2/outbreaks/${id}`, {
      method: "DELETE",
      query: { lock_version: lockVersion },
    });
  },
  transition(
    id: string,
    action: "submit" | "approve" | "publish" | "withdraw",
    input: ServicesTransitionInput,
  ) {
    return transition(
      `/api/v2/outbreaks/${id}/${action}`,
      input,
    ) as Promise<OutbreakRecord>;
  },
  correct(id: string, input: ServicesTransitionInput) {
    return transition(
      `/api/v2/outbreaks/${id}/correct`,
      input,
    ) as Promise<OutbreakRecord>;
  },
  updateStatus(id: string, input: ServicesTransitionInput) {
    return transition(
      `/api/v2/outbreaks/${id}/update-status`,
      input,
    ) as Promise<OutbreakRecord>;
  },
  listUpdates(id: string, page = 1) {
    return client().send<PagedResult<OutbreakUpdateRecord>>(
      `/api/v2/outbreaks/${id}/updates`,
      { query: { page, per_page: 100 } },
    );
  },
  createUpdate(id: string, input: ChildContentInput) {
    return client().send<OutbreakUpdateRecord>(
      `/api/v2/outbreaks/${id}/updates`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  updateUpdate(id: string, childId: string, input: ChildContentInput) {
    return client().send<OutbreakUpdateRecord>(
      `/api/v2/outbreaks/${id}/updates/${childId}`,
      { method: "PATCH", body: JSON.stringify(input) },
    );
  },
  transitionUpdate(
    id: string,
    childId: string,
    action: "submit" | "approve" | "publish" | "withdraw",
    input: ServicesTransitionInput,
  ) {
    return transition(
      `/api/v2/outbreaks/${id}/updates/${childId}/${action}`,
      input,
    ) as Promise<OutbreakUpdateRecord>;
  },
  correctUpdate(id: string, childId: string, input: ServicesTransitionInput) {
    return transition(
      `/api/v2/outbreaks/${id}/updates/${childId}/correct`,
      input,
    ) as Promise<OutbreakUpdateRecord>;
  },
  editPublishedUpdate(id: string, childId: string, input: ServicesResourceCorrectionInput) {
    return client().send<OutbreakUpdateRecord>(
      `/api/v2/outbreaks/${id}/updates/${childId}/correct`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  deleteUpdate(id: string, childId: string, lockVersion: number) {
    return client().send<void>(
      `/api/v2/outbreaks/${id}/updates/${childId}`,
      { method: "DELETE", query: { lock_version: lockVersion } },
    );
  },
  listResources(id: string, page = 1) {
    return client().send<PagedResult<OutbreakResourceRecord>>(
      `/api/v2/outbreaks/${id}/resources`,
      { query: { page, per_page: 100 } },
    );
  },
  createResource(id: string, input: ChildContentInput) {
    return client().send<OutbreakResourceRecord>(
      `/api/v2/outbreaks/${id}/resources`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  updateResource(id: string, childId: string, input: ChildContentInput) {
    return client().send<OutbreakResourceRecord>(
      `/api/v2/outbreaks/${id}/resources/${childId}`,
      { method: "PATCH", body: JSON.stringify(input) },
    );
  },
  transitionResource(
    id: string,
    childId: string,
    action: "submit" | "approve" | "publish" | "withdraw",
    input: ServicesTransitionInput,
  ) {
    return transition(
      `/api/v2/outbreaks/${id}/resources/${childId}/${action}`,
      input,
    ) as Promise<OutbreakResourceRecord>;
  },
  correctResource(id: string, childId: string, input: ServicesTransitionInput) {
    return transition(
      `/api/v2/outbreaks/${id}/resources/${childId}/correct`,
      input,
    ) as Promise<OutbreakResourceRecord>;
  },
  editPublishedResource(id: string, childId: string, input: ServicesResourceCorrectionInput) {
    return client().send<OutbreakResourceRecord>(
      `/api/v2/outbreaks/${id}/resources/${childId}/correct`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  deleteResource(id: string, childId: string, lockVersion: number) {
    return client().send<void>(
      `/api/v2/outbreaks/${id}/resources/${childId}`,
      { method: "DELETE", query: { lock_version: lockVersion } },
    );
  },
  createCampaign(id: string, input: OutbreakCampaignInput) {
    return client().send<ServicesNotificationCampaignDTO>(
      `/api/v2/outbreaks/${id}/notification-campaign`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  audit(id: string, page = 1, perPage = 20) {
    return client().send<PagedResult<OutbreakAuditRecord>>(
      `/api/v2/outbreaks/${id}/audit`,
      { query: { page, per_page: perPage } },
    );
  },
  addReviewComment(id: string, comment: string) {
    return client().send<void>(`/api/v2/outbreaks/${id}/review-comments`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    });
  },
};

export const situationReportsService = {
  list(query: OutbreakListQuery = {}) {
    return client().send<PagedResult<SituationReportRecord>>(
      "/api/v2/situation-reports",
      { query: { ...query } },
    );
  },
  get(id: string) {
    return client().send<SituationReportRecord>(
      `/api/v2/situation-reports/${id}`,
    );
  },
  create(input: SituationReportInput) {
    return client().send<SituationReportRecord>("/api/v2/situation-reports", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  update(id: string, input: SituationReportInput) {
    return client().send<SituationReportRecord>(
      `/api/v2/situation-reports/${id}`,
      { method: "PATCH", body: JSON.stringify(input) },
    );
  },
  remove(id: string, lockVersion: number) {
    return client().send<void>(`/api/v2/situation-reports/${id}`, {
      method: "DELETE",
      query: { lock_version: lockVersion },
    });
  },
  transition(
    id: string,
    action: "submit" | "approve" | "publish" | "withdraw",
    input: ServicesTransitionInput,
  ) {
    return transition(
      `/api/v2/situation-reports/${id}/${action}`,
      input,
    ) as Promise<SituationReportRecord>;
  },
  correct(id: string, input: ServicesTransitionInput) {
    return transition(
      `/api/v2/situation-reports/${id}/correct`,
      input,
    ) as Promise<SituationReportRecord>;
  },
  updateMetrics(id: string, input: OutbreakMetricsInput) {
    return client().send<SituationReportRecord>(
      `/api/v2/situation-reports/${id}/metrics`,
      { method: "PATCH", body: JSON.stringify(input) },
    );
  },
  listAttachments(id: string) {
    return client().send<SituationReportAttachmentRecord[]>(
      `/api/v2/situation-reports/${id}/attachments`,
    );
  },
  createAttachment(id: string, input: SituationReportAttachmentInput) {
    return client().send<SituationReportAttachmentRecord>(
      `/api/v2/situation-reports/${id}/attachments`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  updateAttachment(id: string, attachmentId: string, input: SituationReportAttachmentInput) {
    return client().send<SituationReportAttachmentRecord>(
      `/api/v2/situation-reports/${id}/attachments/${attachmentId}`,
      { method: "PATCH", body: JSON.stringify(input) },
    );
  },
  removeAttachment(id: string, attachmentId: string, lockVersion: number) {
    return client().send<void>(
      `/api/v2/situation-reports/${id}/attachments/${attachmentId}`,
      { method: "DELETE", query: { lock_version: lockVersion } },
    );
  },
  createCampaign(id: string, input: OutbreakCampaignInput) {
    return client().send<ServicesNotificationCampaignDTO>(
      `/api/v2/situation-reports/${id}/notification-campaign`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },
  audit(id: string, page = 1, perPage = 20) {
    return client().send<PagedResult<OutbreakAuditRecord>>(
      `/api/v2/situation-reports/${id}/audit`,
      { query: { page, per_page: perPage } },
    );
  },
  addReviewComment(id: string, comment: string) {
    return client().send<void>(
      `/api/v2/situation-reports/${id}/review-comments`,
      { method: "POST", body: JSON.stringify({ comment }) },
    );
  },
};
