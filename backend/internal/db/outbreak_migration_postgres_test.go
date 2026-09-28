package db

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestOutbreakAdministrationMigrationUpDownUp(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("MEDIGUIDE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("MEDIGUIDE_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, `DROP SCHEMA IF EXISTS "`+schema+`" CASCADE`)

	testDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()
	testDB.SetMaxOpenConns(1)
	testDB.SetMaxIdleConns(1)
	if _, err := testDB.ExecContext(ctx, `SET search_path TO "`+schema+`", public`); err != nil {
		t.Fatal(err)
	}
	var activeSchema string
	if err := testDB.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&activeSchema); err != nil || activeSchema != schema {
		t.Fatalf("isolated migration schema not active: schema=%q err=%v", activeSchema, err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetTableName(schema + ".goose_db_version")
	defer goose.SetTableName("goose_db_version")
	if err := goose.Up(testDB, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	var chunkBlockIndexDefinition string
	if err := testDB.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname = $1 AND indexname = 'idx_guideline_chunks_block_id_fk'`, schema).Scan(&chunkBlockIndexDefinition); err != nil || !strings.Contains(chunkBlockIndexDefinition, "(block_id)") || strings.Contains(strings.ToUpper(chunkBlockIndexDefinition), " WHERE ") {
		t.Fatalf("guideline chunk FK index must cover every block reference: definition=%q err=%v", chunkBlockIndexDefinition, err)
	}
	if err := goose.DownTo(testDB, "../../migrations", 35); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(testDB, "../../migrations", 53); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'guideline_version_manifests' AND column_name IN ('reviewed_section_count','leaf_section_count','reviewed_leaf_section_count','empty_leaf_section_count','reviewed_paragraph_count')`, schema).Scan(&count); err != nil || count != 5 {
		t.Fatalf("guideline completeness columns missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'outbreaks' AND column_name = 'lock_version'`, schema).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbreak lock_version missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname IN ('idx_outbreaks_public_search','idx_situation_reports_public_search')`, schema).Scan(&count); err != nil || count != 2 {
		t.Fatalf("outbreak public search indexes missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.table_constraints WHERE constraint_schema = $1 AND constraint_name IN ('outbreaks_metrics_array_check','situation_reports_highlights_array_check')`, schema).Scan(&count); err != nil || count != 2 {
		t.Fatalf("outbreak JSON safety constraints missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM notification_templates WHERE template_key IN ('outbreak-alert','outbreak-update','outbreak-status-change','situation-report-publication') AND status = 'published'`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("outbreak notification templates missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname = 'idx_outbreaks_title_search'`, schema).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbreak title search index missing after up/down/up: count=%d err=%v", count, err)
	}
	// Outbreak documents were removed: their columns, indexes and templates must be gone.
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'outbreak_resources' AND column_name IN ('storage_key','search_content','document_number','version','content_sections','indexed_at')`, schema).Scan(&count); err != nil || count != 0 {
		t.Fatalf("outbreak document columns remain after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname LIKE 'idx_outbreak_resources_document%'`, schema).Scan(&count); err != nil || count != 0 {
		t.Fatalf("outbreak document indexes remain after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM notification_templates WHERE template_key LIKE 'outbreak-document-%'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("outbreak document notification templates remain after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'calculators' AND column_name IN ('runtime_type','current_version_id')`, schema).Scan(&count); err != nil || count != 2 {
		t.Fatalf("calculator version columns missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name IN ('calculator_versions','calculator_test_cases','calculator_citations','calculator_version_audits')`, schema).Scan(&count); err != nil || count != 4 {
		t.Fatalf("calculator version tables missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname = 'idx_calculator_versions_one_published'`, schema).Scan(&count); err != nil || count != 1 {
		t.Fatalf("single-published-version index missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name IN ('diseases','disease_aliases','disease_codes','disease_taxonomy_migration_report')`, schema).Scan(&count); err != nil || count != 4 {
		t.Fatalf("disease taxonomy tables missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM diseases WHERE status = 'active' AND deleted_at IS NULL`).Scan(&count); err != nil || count != 7 {
		t.Fatalf("initial disease taxonomy seed mismatch: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name IN ('guideline_document_categories','content_disease_assignments')`, schema).Scan(&count); err != nil || count != 2 {
		t.Fatalf("content classification tables missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = $1 AND indexname IN ('idx_guideline_document_categories_category','idx_content_disease_assignment_unique','idx_content_disease_assignment_primary','idx_content_disease_assignment_disease','idx_content_disease_assignment_resource')`, schema).Scan(&count); err != nil || count != 5 {
		t.Fatalf("content classification indexes missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name IN ('content_hubs','content_hub_diseases','content_pillars','content_pillar_items','content_hub_templates','content_hub_template_pillars')`, schema).Scan(&count); err != nil || count != 6 {
		t.Fatalf("content hub tables missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM content_hub_templates WHERE status = 'active' AND deleted_at IS NULL`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("content hub template seed mismatch: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM content_hub_template_pillars WHERE deleted_at IS NULL`).Scan(&count); err != nil || count != 32 {
		t.Fatalf("content hub template pillar seed mismatch: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM permissions WHERE code IN (
		'disease.taxonomy.read','disease.taxonomy.manage','disease.assignment.read','disease.assignment.manage',
		'content_hub.read','content_hub.manage','content_hub.publish','content_hub.archive',
		'content_pillar.read','content_pillar.manage','content_hub.template.read','content_hub.template.manage'
	) AND deleted_at IS NULL`).Scan(&count); err != nil || count != 12 {
		t.Fatalf("disease/hub permissions missing after up/down/up: count=%d err=%v", count, err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT count(*) FROM role_permissions rp JOIN roles r ON r.id = rp.role_id JOIN permissions p ON p.id = rp.permission_id WHERE lower(coalesce(r.role_key, r.name)) IN ('content_manager','editor') AND p.code IN ('content_hub.publish','content_hub.archive')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("ordinary editors received hub publication permissions: count=%d err=%v", count, err)
	}
	var pillarItemDefault string
	if err := testDB.QueryRowContext(ctx, `SELECT column_default FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'content_pillar_items' AND column_name = 'status'`, schema).Scan(&pillarItemDefault); err != nil || !strings.Contains(pillarItemDefault, "draft") {
		t.Fatalf("content pillar items do not default to draft: default=%q err=%v", pillarItemDefault, err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO diseases (id, name, normalized_name, slug) VALUES
		('92000000-0000-4000-8000-000000000001', 'Cycle parent', 'cycle parent', 'cycle-parent'),
		('92000000-0000-4000-8000-000000000002', 'Cycle child', 'cycle child', 'cycle-child')`); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE diseases SET parent_id = '92000000-0000-4000-8000-000000000001' WHERE id = '92000000-0000-4000-8000-000000000002'`); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE diseases SET parent_id = '92000000-0000-4000-8000-000000000002' WHERE id = '92000000-0000-4000-8000-000000000001'`); err == nil {
		t.Fatal("PostgreSQL disease hierarchy trigger accepted a cycle")
	}
}
