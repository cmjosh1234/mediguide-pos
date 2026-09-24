package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

// Goose panics at startup when two files share a version, which parallel
// branches cause easily; catch it here instead of in a running container.
func TestMigrationVersionsAreUnique(t *testing.T) {
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no migrations found")
	}
	seen := map[int64]string{}
	for _, file := range files {
		version, err := goose.NumericComponent(file)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(file), err)
		}
		if previous, ok := seen[version]; ok {
			t.Errorf("duplicate migration version %d: %s and %s; renumber the newer one", version, previous, filepath.Base(file))
		}
		seen[version] = filepath.Base(file)
	}
}

func TestNotificationTemplateLegacyVariables(t *testing.T) {
	dsn := os.Getenv("MEDIGUIDE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MEDIGUIDE_TEST_DATABASE_URL is not configured")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema := "migration_" + uuid.New().String()
	if _, err := db.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	if _, err := db.Exec(`SET search_path TO "` + schema + `", public`); err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetTableName(`"` + schema + `".goose_db_version`)
	defer goose.SetTableName("goose_db_version")
	if err := goose.UpTo(db, "../../migrations", 30); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		input any
		want  string
	}{
		{"legacy names", `["guideline_name","protocol_name"]`, `{"guideline_name":{"type":"string","required":true,"sample_value":"guideline_name"},"protocol_name":{"type":"string","required":true,"sample_value":"protocol_name"}}`},
		{"type map", `{"count":"number"}`, `{"count":{"type":"number","required":true,"sample_value":"count"}}`},
		{"schema map", `{"name":{"type":"string","required":false,"sample_value":"Example"}}`, `{"name":{"type":"string","required":false,"sample_value":"Example"}}`},
		{"SQL null", nil, `{}`},
		{"JSON null", `null`, `{}`},
		{"empty array", `[]`, `{}`},
		{"empty object", `{}`, `{}`},
		{"scalar", `42`, `{}`},
	}
	for _, tc := range cases {
		if _, err := db.Exec(`INSERT INTO notification_templates
			(name, type, category, status, content, variables_json)
			VALUES ($1, 'push', 'system', 'active', 'Legacy message', $2::jsonb)`, tc.name, tc.input); err != nil {
			t.Fatal(err)
		}
	}
	if err := goose.UpTo(db, "../../migrations", 31); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var matches bool
			err := db.QueryRow(`SELECT v.variable_schema_json = $2::jsonb
				AND v.status = 'published' AND v.body_template = 'Legacy message'
				FROM notification_template_versions v
				JOIN notification_templates t ON t.id = v.template_id
				WHERE t.name = $1`, tc.name, tc.want).Scan(&matches)
			if err != nil || !matches {
				t.Fatalf("migrated template does not match expected schema %s: %v", tc.want, err)
			}
		})
	}
}
