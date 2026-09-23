package main

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ucgDiseaseTaxonomyCSV is the candidate taxonomy extracted from the Uganda
// Clinical Guidelines by extract_ucg_taxonomy.py. Every name and every ICD-10
// code is the guideline's own; nothing here is inferred.
//
// The seed_action column carries the review decision for each row, so striking
// an entry that is not a clinical subject, or restoring one, is an edit to that
// file rather than a change to the code below.
//
//go:embed fixtures/ucg-disease-taxonomy.csv
var ucgDiseaseTaxonomyCSV string

const ucgTaxonomyNamespace = "mediguide/seed/ucg-disease-taxonomy/"

// ucgTaxonomyID derives a stable identifier for a taxonomy row so that reruns
// update rather than duplicate. It uses its own namespace rather than demoID:
// this taxonomy is reference data and must not share identifiers with the
// demo-only fixtures.
func ucgTaxonomyID(kind, key string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(ucgTaxonomyNamespace+kind+"/"+key))
}

// ucgCuratedDiseaseIDs lists the slugs that migration 00048 already owns and
// that the UCG names again. The unique index on an active normalized name means
// these must be reused, not inserted a second time. The seed leaves their row
// untouched, because name, description, icon, colour and sort order are curated
// there, and only hangs UCG children, aliases and codes off the existing id.
var ucgCuratedDiseaseIDs = map[string]uuid.UUID{
	"cholera":           uuid.MustParse("90000000-0000-4000-8000-000000000003"),
	"diabetes-mellitus": uuid.MustParse("90000000-0000-4000-8000-000000000006"),
	"hypertension":      uuid.MustParse("90000000-0000-4000-8000-000000000007"),
	"malaria":           uuid.MustParse("90000000-0000-4000-8000-000000000002"),
	"measles":           uuid.MustParse("90000000-0000-4000-8000-000000000005"),
}

const ucgCodeSystem = "ICD-10"

// utf8BOM is stripped from the fixture before parsing: a spreadsheet round
// trip can reintroduce it, and it would otherwise land in the first column
// name and break the header lookup.
var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

type ucgTaxonomyRow struct {
	Chapter       int
	ChapterTitle  string
	SectionNumber string
	Depth         int
	Name          string
	ShortName     string
	Aliases       []string
	Slug          string
	ParentName    string
	Codes         []string
	Action        string
}

type ucgPlannedDisease struct {
	ID             uuid.UUID
	ParentID       *uuid.UUID
	Name           string
	NormalizedName string
	Slug           string
	ShortName      string
	SortOrder      int
	// Curated marks a disease that already exists from migration 00048. Its
	// diseases row is left alone; only children, aliases and codes are seeded.
	Curated bool
}

type ucgPlannedAlias struct {
	ID              uuid.UUID
	DiseaseID       uuid.UUID
	Alias           string
	NormalizedAlias string
}

type ucgPlannedCode struct {
	ID         uuid.UUID
	DiseaseID  uuid.UUID
	CodeSystem string
	Code       string
	// DisplayName is left empty for a code on a curated disease. Those rows
	// already carry a curated label, such as "Essential hypertension" on I10,
	// and the guideline heading would be a downgrade.
	DisplayName string
}

// ucgPlannedCategory is a guideline category made from a UCG chapter. Its ID
// is only used when no category with the slug exists yet; an existing one is
// reused as it stands.
type ucgPlannedCategory struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	SortOrder int
}

type ucgPlannedCategoryLink struct {
	DiseaseID    uuid.UUID
	CategorySlug string
}

type ucgTaxonomyPlan struct {
	// Diseases is ordered parents before children, which the hierarchy trigger
	// requires: it rejects a parent row that does not yet exist or is not active.
	Diseases []ucgPlannedDisease
	Aliases  []ucgPlannedAlias
	Codes    []ucgPlannedCode
	// Categories holds one category per guideline chapter, in chapter order.
	Categories []ucgPlannedCategory
	// CategoryLinks files every disease under the chapter it appears in. A
	// subject the guideline repeats under two chapters is filed under both.
	CategoryLinks []ucgPlannedCategoryLink
	// Notes records every reconciliation the planner had to make, so that a run
	// can be checked against the review document.
	Notes []string
}

// normalizeUCGTerm mirrors normalizeDiseaseTerm in
// internal/services/disease_service.go. The seed writes normalized_name and
// normalized_alias directly, so the two must agree or lookups made through the
// service will miss these rows.
func normalizeUCGTerm(value string) string {
	var out strings.Builder
	space := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out.WriteRune(r)
			space = false
		} else if !space {
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(out.String())
}

func parseUCGTaxonomyRows(data string) ([]ucgTaxonomyRow, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(data, utf8BOM)))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read taxonomy fixture: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("taxonomy fixture is empty")
	}

	column := map[string]int{}
	for index, name := range records[0] {
		column[strings.TrimSpace(name)] = index
	}
	for _, required := range []string{
		"chapter", "chapter_title", "section_number", "depth", "name", "short_name",
		"aliases", "slug", "parent_name", "icd10_codes", "seed_action",
	} {
		if _, ok := column[required]; !ok {
			return nil, fmt.Errorf("taxonomy fixture is missing the %q column", required)
		}
	}
	field := func(record []string, name string) string {
		index := column[name]
		if index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}

	rows := make([]ucgTaxonomyRow, 0, len(records)-1)
	for number, record := range records[1:] {
		chapter, err := strconv.Atoi(field(record, "chapter"))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid chapter: %w", number+2, err)
		}
		depth, err := strconv.Atoi(field(record, "depth"))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid depth: %w", number+2, err)
		}
		action := field(record, "seed_action")
		if action != "seed" && action != "skip" {
			return nil, fmt.Errorf("row %d: seed_action must be seed or skip, got %q", number+2, action)
		}
		rows = append(rows, ucgTaxonomyRow{
			Chapter:       chapter,
			ChapterTitle:  field(record, "chapter_title"),
			SectionNumber: field(record, "section_number"),
			Depth:         depth,
			Name:          field(record, "name"),
			ShortName:     field(record, "short_name"),
			Aliases:       splitUCGList(field(record, "aliases"), "|"),
			Slug:          field(record, "slug"),
			ParentName:    field(record, "parent_name"),
			Codes:         splitUCGList(field(record, "icd10_codes"), " "),
			Action:        action,
		})
	}
	return rows, nil
}

func splitUCGList(value, separator string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, separator)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// ucgCategorySlug derives a category slug from a chapter title, such as
// "family-planning-fp" from "Family Planning (FP)".
func ucgCategorySlug(title string) string {
	return strings.ReplaceAll(normalizeUCGTerm(title), " ", "-")
}

// planUCGDiseaseTaxonomy turns the reviewed rows into the exact set of records
// to write. It is kept separate from the database work so the reconciliation
// rules can be tested directly.
//
// Four rules reconcile the guideline against the hard constraints of the schema:
//
//  1. A slug is globally unique, so where the UCG repeats a subject under two
//     chapters the first occurrence wins and the later one merges into it,
//     carrying over its aliases, codes and children.
//  2. A row struck in review is not seeded. A surviving child of a struck parent
//     is re-parented to the root, because the guideline section above it is a
//     grouping rather than a disease.
//  3. An ICD-10 code maps to at most one disease, so where the UCG repeats a
//     code the first claimant keeps it and later claims are dropped.
//  4. A disease already owned by migration 00048 keeps its curated row.
func planUCGDiseaseTaxonomy(rows []ucgTaxonomyRow) (ucgTaxonomyPlan, error) {
	plan := ucgTaxonomyPlan{}

	// Rule 1: the first occurrence of a slug wins and later rows merge into it.
	idBySlug := map[string]uuid.UUID{}
	slugByName := map[string]string{}
	winners := make([]ucgTaxonomyRow, 0, len(rows))
	for _, row := range rows {
		if row.Action != "seed" {
			continue
		}
		if row.Slug == "" || row.Name == "" {
			return plan, fmt.Errorf("section %s: name and slug are required", row.SectionNumber)
		}
		if _, seen := idBySlug[row.Slug]; seen {
			plan.Notes = append(plan.Notes, fmt.Sprintf(
				"merged %s %q into the earlier row with slug %q", row.SectionNumber, row.Name, row.Slug))
			continue
		}
		id, curated := ucgCuratedDiseaseIDs[row.Slug]
		if !curated {
			id = ucgTaxonomyID("disease", row.Slug)
		}
		idBySlug[row.Slug] = id
		if _, exists := slugByName[row.Name]; !exists {
			slugByName[row.Name] = row.Slug
		}
		winners = append(winners, row)
	}

	// Rule 2: resolve each parent, falling back to the root when the parent was
	// struck in review.
	parentOf := map[string]string{}
	for _, row := range winners {
		if row.ParentName == "" {
			continue
		}
		parentSlug, ok := slugByName[row.ParentName]
		if !ok {
			plan.Notes = append(plan.Notes, fmt.Sprintf(
				"re-parented %s %q to the root: its parent %q is not seeded",
				row.SectionNumber, row.Name, row.ParentName))
			continue
		}
		if parentSlug == row.Slug {
			return plan, fmt.Errorf("section %s: %q is its own parent", row.SectionNumber, row.Name)
		}
		parentOf[row.Slug] = parentSlug
	}

	// Sort order groups the roots by guideline chapter and keeps every sibling
	// set in the order the guideline presents it.
	rootsPerChapter := map[int]int{}
	childrenPerParent := map[string]int{}
	for _, row := range winners {
		id := idBySlug[row.Slug]
		_, curated := ucgCuratedDiseaseIDs[row.Slug]

		var parentID *uuid.UUID
		sortOrder := 0
		if parentSlug, ok := parentOf[row.Slug]; ok {
			resolved := idBySlug[parentSlug]
			parentID = &resolved
			childrenPerParent[parentSlug] += 10
			sortOrder = childrenPerParent[parentSlug]
		} else {
			rootsPerChapter[row.Chapter] += 10
			sortOrder = row.Chapter*1000 + rootsPerChapter[row.Chapter]
		}

		plan.Diseases = append(plan.Diseases, ucgPlannedDisease{
			ID:             id,
			ParentID:       parentID,
			Name:           row.Name,
			NormalizedName: normalizeUCGTerm(row.Name),
			Slug:           row.Slug,
			ShortName:      row.ShortName,
			SortOrder:      sortOrder,
			Curated:        curated,
		})
	}

	// The hierarchy trigger rejects a parent that does not exist yet, so every
	// root must be written before any child. Curated rows are never written, but
	// they already exist, which makes them equally safe as parents.
	sort.SliceStable(plan.Diseases, func(i, j int) bool {
		return plan.Diseases[i].ParentID == nil && plan.Diseases[j].ParentID != nil
	})

	// Aliases carry over from merged rows too. An alias that collides with a
	// seeded disease name is dropped, because validateDiseaseIdentity would
	// later reject it as ambiguous.
	normalizedNames := map[string]bool{}
	for _, disease := range plan.Diseases {
		normalizedNames[disease.NormalizedName] = true
	}
	seenAlias := map[string]bool{}
	for _, row := range rows {
		if row.Action != "seed" {
			continue
		}
		diseaseID, ok := idBySlug[row.Slug]
		if !ok {
			continue
		}
		for _, alias := range row.Aliases {
			normalized := normalizeUCGTerm(alias)
			if normalized == "" {
				continue
			}
			if normalizedNames[normalized] {
				plan.Notes = append(plan.Notes, fmt.Sprintf(
					"dropped alias %q on %q: it matches a seeded disease name", alias, row.Name))
				continue
			}
			key := diseaseID.String() + "/" + normalized
			if seenAlias[key] {
				continue
			}
			seenAlias[key] = true
			plan.Aliases = append(plan.Aliases, ucgPlannedAlias{
				ID:              ucgTaxonomyID("disease-alias", row.Slug+"/"+normalized),
				DiseaseID:       diseaseID,
				Alias:           alias,
				NormalizedAlias: normalized,
			})
		}
	}

	// Rule 3: an ICD-10 code maps to at most one disease across the whole table.
	claimedCode := map[string]string{}
	for _, row := range rows {
		if row.Action != "seed" {
			continue
		}
		diseaseID, ok := idBySlug[row.Slug]
		if !ok {
			continue
		}
		for _, code := range row.Codes {
			key := strings.ToLower(code)
			if owner, claimed := claimedCode[key]; claimed {
				if owner != row.Slug {
					plan.Notes = append(plan.Notes, fmt.Sprintf(
						"dropped ICD-10 %s from %s %q: already claimed by %q",
						code, row.SectionNumber, row.Name, owner))
				}
				continue
			}
			claimedCode[key] = row.Slug
			displayName := row.Name
			if _, curated := ucgCuratedDiseaseIDs[row.Slug]; curated {
				displayName = ""
			}
			plan.Codes = append(plan.Codes, ucgPlannedCode{
				ID:          ucgTaxonomyID("disease-code", ucgCodeSystem+"/"+key),
				DiseaseID:   diseaseID,
				CodeSystem:  ucgCodeSystem,
				Code:        code,
				DisplayName: displayName,
			})
		}
	}

	// Categories follow the guideline chapters. Every seeded row is filed under
	// its own chapter, including a row merged into an earlier one by rule 1.
	titleByChapter := map[int]string{}
	chapters := []int{}
	seenLink := map[string]bool{}
	for _, row := range rows {
		if row.Action != "seed" {
			continue
		}
		if row.ChapterTitle == "" {
			return plan, fmt.Errorf("section %s: chapter title is required", row.SectionNumber)
		}
		if title, seen := titleByChapter[row.Chapter]; !seen {
			titleByChapter[row.Chapter] = row.ChapterTitle
			chapters = append(chapters, row.Chapter)
		} else if title != row.ChapterTitle {
			return plan, fmt.Errorf("chapter %d is titled both %q and %q", row.Chapter, title, row.ChapterTitle)
		}
		diseaseID, ok := idBySlug[row.Slug]
		if !ok {
			continue
		}
		categorySlug := ucgCategorySlug(row.ChapterTitle)
		key := diseaseID.String() + "/" + categorySlug
		if seenLink[key] {
			continue
		}
		seenLink[key] = true
		plan.CategoryLinks = append(plan.CategoryLinks, ucgPlannedCategoryLink{
			DiseaseID:    diseaseID,
			CategorySlug: categorySlug,
		})
	}
	sort.Ints(chapters)
	for _, chapter := range chapters {
		slug := ucgCategorySlug(titleByChapter[chapter])
		plan.Categories = append(plan.Categories, ucgPlannedCategory{
			ID:        ucgTaxonomyID("category", slug),
			Name:      titleByChapter[chapter],
			Slug:      slug,
			SortOrder: chapter * 10,
		})
	}

	sort.Strings(plan.Notes)
	return plan, nil
}

// seedUCGDiseaseTaxonomy writes the reviewed Uganda Clinical Guidelines
// taxonomy. It is idempotent: identifiers are derived from the slug and the
// code, so a rerun updates the same rows. actorID is recorded as the author
// when it is set.
func seedUCGDiseaseTaxonomy(database *gorm.DB, actorID uuid.UUID) error {
	rows, err := parseUCGTaxonomyRows(ucgDiseaseTaxonomyCSV)
	if err != nil {
		return err
	}
	plan, err := planUCGDiseaseTaxonomy(rows)
	if err != nil {
		return err
	}

	written := 0
	for _, disease := range plan.Diseases {
		if disease.Curated {
			continue
		}
		row := map[string]any{
			"id":              disease.ID,
			"parent_id":       disease.ParentID,
			"name":            disease.Name,
			"normalized_name": disease.NormalizedName,
			"slug":            disease.Slug,
			"status":          "active",
			"sort_order":      disease.SortOrder,
			"deleted_at":      nil,
		}
		if disease.ShortName != "" {
			row["short_name"] = disease.ShortName
		}
		if actorID != uuid.Nil {
			row["created_by"] = actorID
			row["updated_by"] = actorID
		}
		if err := upsertByID(database, "diseases", row); err != nil {
			return fmt.Errorf("seed disease %q: %w", disease.Slug, err)
		}
		written++
	}

	for _, alias := range plan.Aliases {
		row := map[string]any{
			"id":               alias.ID,
			"disease_id":       alias.DiseaseID,
			"alias":            alias.Alias,
			"normalized_alias": alias.NormalizedAlias,
			"deleted_at":       nil,
		}
		if err := upsertByID(database, "disease_aliases", row); err != nil {
			return fmt.Errorf("seed alias %q: %w", alias.Alias, err)
		}
	}

	for _, code := range plan.Codes {
		row := map[string]any{
			"id":          code.ID,
			"disease_id":  code.DiseaseID,
			"code_system": code.CodeSystem,
			"code":        code.Code,
			"deleted_at":  nil,
		}
		if code.DisplayName != "" {
			row["display_name"] = code.DisplayName
		}
		if err := upsertByID(database, "disease_codes", row); err != nil {
			return fmt.Errorf("seed code %s %s: %w", code.CodeSystem, code.Code, err)
		}
	}

	// A category that already exists under the slug, whether curated by an
	// administrator or seeded by the demo, is reused and left untouched.
	categoryIDs := map[string]uuid.UUID{}
	createdCategories := 0
	for _, category := range plan.Categories {
		existingID, found, err := lookupRowIDByColumn(database, "guideline_categories", "slug", category.Slug)
		if err != nil {
			return fmt.Errorf("look up category %q: %w", category.Slug, err)
		}
		if found {
			categoryIDs[category.Slug] = existingID
			continue
		}
		row := map[string]any{
			"id":         category.ID,
			"name":       category.Name,
			"slug":       category.Slug,
			"sort_order": category.SortOrder,
			"status":     "active",
			"deleted_at": nil,
		}
		if err := upsertByID(database, "guideline_categories", row); err != nil {
			return fmt.Errorf("seed category %q: %w", category.Slug, err)
		}
		categoryIDs[category.Slug] = category.ID
		createdCategories++
	}

	for _, link := range plan.CategoryLinks {
		row := models.DiseaseCategory{DiseaseID: link.DiseaseID, CategoryID: categoryIDs[link.CategorySlug]}
		if err := database.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return fmt.Errorf("file disease %s under %q: %w", link.DiseaseID, link.CategorySlug, err)
		}
	}

	for _, note := range plan.Notes {
		log.Info().Str("seed", "disease-taxonomy").Msg(note)
	}
	log.Info().
		Int("candidates", len(rows)).
		Int("diseases_written", written).
		Int("diseases_curated", len(plan.Diseases)-written).
		Int("aliases", len(plan.Aliases)).
		Int("codes", len(plan.Codes)).
		Int("categories", len(plan.Categories)).
		Int("categories_created", createdCategories).
		Int("category_links", len(plan.CategoryLinks)).
		Int("reconciliations", len(plan.Notes)).
		Msg("UCG disease taxonomy seeded")
	return nil
}
