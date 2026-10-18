package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"mediguide/internal/models"
	"mediguide/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type demoWorkflowRow struct {
	table  string
	values map[string]any
}

const demoWorkflowSyncManifest = `{"demo":true,"version":"1.0.0","description":"Synthetic development sync package; no clinical content","guidelines":[]}`

func demoWorkflowID(table string) uuid.UUID { return demoID("workflow-history", table) }

// These are synthetic development histories, not requests to workers. Tokens
// are consumed, devices opt out, and every dispatch job is already terminal.
// Called only by the production-guarded demo scope, after its parent fixtures.
func seedDemoWorkflowHistory(ctx context.Context, database *gorm.DB, store storage.ObjectStore, adminID, clinicianID uuid.UUID) error {
	var subcounty models.Subcounty
	if err := database.Order("id ASC").First(&subcounty).Error; err != nil {
		return fmt.Errorf("workflow fixtures require the facility seed: %w", err)
	}
	var source models.GuidelineVersion
	if err := database.First(&source, "id = ?", demoID("guideline-version", "reader-test-form-1.0")).Error; err != nil {
		return fmt.Errorf("workflow fixtures require the reader form: %w", err)
	}
	var sourceAsset models.GuidelineAsset
	if err := database.First(&sourceAsset, "version_id = ? AND type = ?", source.ID, "original_pdf").Error; err != nil {
		return fmt.Errorf("workflow fixtures require the reader form original: %w", err)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(file, demoWorkflowSyncManifest); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	key := "development-seed/workflow-history/sync-package.zip"
	if err := store.Put(ctx, key, bytes.NewReader(archive.Bytes()), int64(archive.Len()), "application/zip"); err != nil {
		return err
	}
	rows := demoWorkflowHistoryRows(adminID, clinicianID, subcounty.ID, source, sourceAsset, key, int64(archive.Len()), demoWorkflowChecksum(archive.Bytes()))
	for _, row := range rows {
		if err := upsertByID(database, row.table, row.values); err != nil {
			return fmt.Errorf("seed workflow history %s: %w", row.table, err)
		}
	}
	return nil
}

func demoWorkflowChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func demoWorkflowHistoryRows(adminID, clinicianID, subcountyID uuid.UUID, source models.GuidelineVersion, sourceAsset models.GuidelineAsset, packageKey string, packageSize int64, packageChecksum string) []demoWorkflowRow {
	at := time.Date(2026, time.October, 13, 9, 0, 0, 0, time.UTC)
	id := demoWorkflowID
	definition := `{"schema_version":"1.0","tool_type":"checklist","title":"[Demo] Training checklist","description":"Synthetic development fixture, not clinical guidance.","version":"1.0.0","locale":"en","warnings":[],"citations":[],"inputs":[{"key":"reviewed","type":"checklist_item","label":"Review sample","required":true,"section_key":"training","checklist_kind":"action"}],"sections":[{"key":"training","title":"Training","order":0}],"calculation":[],"rules":[],"outputs":[],"interpretations":[],"completion":{"mode":"all_required","allow_resume":true,"require_review":false,"show_percentage":true,"reset_confirmation":true},"test_cases":[]}`
	row := func(table string, values map[string]any) demoWorkflowRow {
		values["id"] = id(table)
		values["created_at"] = at
		// Account tokens and emails intentionally have no soft-delete column.
		if table != "account_action_tokens" {
			values["updated_at"] = at
		}
		if table != "account_action_tokens" && table != "account_email_deliveries" {
			values["deleted_at"] = nil
		}
		return demoWorkflowRow{table, values}
	}
	return []demoWorkflowRow{
		row("parishes", map[string]any{"subcounty_id": subcountyID, "name": "[Demo] Training parish", "nhpi_code": "DEMO-WORKFLOW-PARISH", "hsdt_code": "DEMO-WORKFLOW-PARISH"}),
		row("account_action_tokens", map[string]any{"user_id": clinicianID, "purpose": "email_verification", "token_hash": demoWorkflowChecksum([]byte("consumed-development-fixture")), "expires_at": at.Add(time.Hour), "consumed_at": at}),
		row("account_email_deliveries", map[string]any{"user_id": clinicianID, "token_id": id("account_action_tokens"), "purpose": "email_verification", "encrypted_token": "", "status": "cancelled", "attempts": 0, "next_attempt_at": at, "lease_until": nil, "sent_at": nil, "last_error_code": "demo_fixture_no_send"}),
		// Own a draft tool so no citation or synthetic definition is attached to
		// a published, clinically reviewed calculator.
		row("calculators", map[string]any{"added_by_user_id": adminID, "name": "[Demo] Training checklist", "description": "Synthetic workflow fixture; not clinical guidance.", "app_file_json": mustJSON(`{}`), "version": "1.0.0", "type": "checklist", "runtime_type": "schema_v1", "status": "draft"}),
		row("calculator_versions", map[string]any{"calculator_id": id("calculators"), "semantic_version": "1.0.0", "schema_version": "1.0", "definition_json": mustJSON(definition), "definition_checksum": demoWorkflowChecksum([]byte(definition)), "status": "draft", "created_by": adminID, "change_summary": "Synthetic local development fixture", "lock_version": 1}),
		row("calculator_citations", map[string]any{"calculator_version_id": id("calculator_versions"), "citation_key": "demo-training", "title": "[Demo] Training reference placeholder", "organization": "MediGuide development fixtures", "url": "https://example.org", "sort_order": 1}),
		row("clinical_protocols", map[string]any{"code": "demo-workflow-training", "title": "[Demo] Training protocol", "program_area": "Development testing", "version": "1.0.0", "language": "en", "status": "draft", "definition_json": mustJSON(`{"demo":true,"steps":[]}`), "definition_yaml": "demo: true\nsteps: []\n"}),
		row("protocol_runs", map[string]any{"protocol_id": id("clinical_protocols"), "user_id": clinicianID, "input_json": mustJSON(`{"demo":true}`), "output_json": mustJSON(`{"demo":true,"completed":true,"message":"Synthetic training run; no clinical recommendation"}`)}),
		row("retrieval_logs", map[string]any{"user_id": clinicianID, "question": "[Demo] Where is the sample reader form?", "query_json": mustJSON(`{"demo":true,"query":"Local test - Form"}`), "results_json": mustJSON(fmt.Sprintf(`[{"guideline_id":%q,"title":"Local test - Form","demo":true}]`, source.DocumentID.String()))}),
		row("guideline_uploads", map[string]any{"version_id": source.ID, "user_id": adminID, "filename": "reader-test-form.pdf", "size_bytes": sourceAsset.SizeBytes, "checksum": sourceAsset.Checksum, "object_key": sourceAsset.StorageKey, "multipart_id": "", "part_size": 5 * 1024 * 1024, "status": "completed", "notified_status": "completed", "expires_at": at}),
		row("sync_packages", map[string]any{"name": "[Demo] Empty training sync package", "version": "1.0.0", "status": "published", "file_key": packageKey, "manifest_json": mustJSON(demoWorkflowSyncManifest), "checksum": packageChecksum, "size_bytes": packageSize}),
		row("device_registrations", map[string]any{"device_id": "demo-workflow-installation", "user_id": clinicianID, "platform": "android", "app_version": "demo", "last_sync_at": at.Format(time.RFC3339)}),
		row("device_sync_logs", map[string]any{"device_id": "demo-workflow-installation", "package_id": id("sync_packages"), "status": "completed"}),
		row("firebase_devices", map[string]any{"user_id": clinicianID, "installation_id": "demo-workflow-installation", "registration_token": "demo-workflow-disabled-not-a-push-token", "platform": "android", "app_version": "demo", "locale": "en", "notifications_enabled": false, "last_seen_at": at}),
		row("notification_campaigns", map[string]any{"name": "[Demo] Completed in-app workflow history", "type": "announcement", "status": "completed", "channels_json": mustJSON(`["in-app"]`), "requested_channels_json": mustJSON(`["in-app"]`), "rendered_title": "[Demo] Training completed", "rendered_body": "Synthetic in-app history; no external message was sent.", "resolved_recipient_count": 1, "idempotency_key": "demo:workflow-history:campaign", "lock_version": 1}),
		row("notification_campaign_recipients", map[string]any{"campaign_id": id("notification_campaigns"), "user_id": clinicianID, "status": "completed"}),
		row("notification_digests", map[string]any{"firebase_device_id": id("firebase_devices"), "window_at": at, "status": "accepted", "next_attempt_at": at, "attempt_count": 0}),
		row("notification_outbox_jobs", map[string]any{"campaign_id": id("notification_campaigns"), "recipient_id": id("notification_campaign_recipients"), "user_id": clinicianID, "channel": "in-app", "status": "accepted", "idempotency_key": "demo:workflow-history:outbox", "payload_json": mustJSON(`{"demo":true,"title":"[Demo] Training completed"}`), "attempt_count": 1, "max_attempts": 1, "next_attempt_at": at, "accepted_at": at, "completed_at": at}),
		row("notification_delivery_attempts", map[string]any{"outbox_job_id": id("notification_outbox_jobs"), "attempt_number": 1, "outcome": "accepted", "duration_ms": 0}),
		row("notification_deliveries", map[string]any{"campaign_id": id("notification_campaigns"), "outbox_job_id": id("notification_outbox_jobs"), "notification_id": demoID("notification", "offline-reminder"), "user_id": clinicianID, "channel": "in-app", "state": "opened", "attempt_count": 1, "attempted_at": at, "accepted_at": at, "delivered_at": at, "opened_at": at}),
		row("notification_delivery_events", map[string]any{"delivery_id": id("notification_deliveries"), "user_id": clinicianID, "event_id": "demo-workflow-history-opened", "event_type": "opened", "occurred_at": at}),
		row("notification_topic_jobs", map[string]any{"topic": "public-demo-workflow", "source_type": "guideline", "source_id": source.DocumentID, "status": "failed", "idempotency_key": "demo:workflow-history:topic", "payload_json": mustJSON(`{"demo":true,"title":"[Demo] Broadcast disabled"}`), "attempt_count": 0, "max_attempts": 1, "next_attempt_at": at, "completed_at": at, "last_error_code": "demo_fixture_no_send", "last_error_message": "Synthetic terminal fixture; broadcast intentionally disabled"}),
		row("situation_report_attachments", map[string]any{"situation_report_id": demoID("situation-report", "demo-cholera-kampala-2026-08-18"), "title": "[Demo] Reader form attachment", "description": "Synthetic local reader form for attachment navigation testing.", "issuing_organization": "MediGuide development fixtures", "document_kind": "form", "url": "/public/guidelines/" + source.DocumentID.String(), "sort_order": 1, "lock_version": 1}),
	}
}
