package main

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The fixture is reviewed data, so these totals are asserted exactly: a change
// to any of them means the candidate list moved and should be re-reviewed
// rather than silently reseeded.
const (
	ucgExpectedCandidates = 470
	ucgExpectedSeedRows   = 396
	ucgExpectedDiseases   = 392
	ucgExpectedCurated    = 5
	ucgExpectedRoots      = 313
	ucgExpectedChildren   = 79
	ucgExpectedAliases    = 44
	ucgExpectedCodes      = 281
	ucgExpectedNotes      = 29
	ucgExpectedCategories = 24
	// Two subjects appear under two chapters, so they carry two links each.
	ucgExpectedCategoryLinks = 394
)

func ucgTestPlan(t *testing.T) (ucgTaxonomyPlan, []ucgTaxonomyRow) {
	t.Helper()
	rows, err := parseUCGTaxonomyRows(ucgDiseaseTaxonomyCSV)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	plan, err := planUCGDiseaseTaxonomy(rows)
	if err != nil {
		t.Fatalf("plan taxonomy: %v", err)
	}
	return plan, rows
}

func TestUCGTaxonomyFixtureTotals(t *testing.T) {
	plan, rows := ucgTestPlan(t)

	seeded := 0
	for _, row := range rows {
		if row.Action == "seed" {
			seeded++
		}
	}

	curated, roots, children := 0, 0, 0
	for _, disease := range plan.Diseases {
		if disease.Curated {
			curated++
		}
		if disease.ParentID == nil {
			roots++
		} else {
			children++
		}
	}

	for _, check := range []struct {
		name string
		got  int
		want int
	}{
		{"candidates", len(rows), ucgExpectedCandidates},
		{"rows marked seed", seeded, ucgExpectedSeedRows},
		{"planned diseases", len(plan.Diseases), ucgExpectedDiseases},
		{"curated diseases", curated, ucgExpectedCurated},
		{"roots", roots, ucgExpectedRoots},
		{"children", children, ucgExpectedChildren},
		{"aliases", len(plan.Aliases), ucgExpectedAliases},
		{"codes", len(plan.Codes), ucgExpectedCodes},
		{"reconciliation notes", len(plan.Notes), ucgExpectedNotes},
		{"categories", len(plan.Categories), ucgExpectedCategories},
		{"category links", len(plan.CategoryLinks), ucgExpectedCategoryLinks},
	} {
		if check.got != check.want {
			t.Errorf("%s = %d, want %d", check.name, check.got, check.want)
		}
	}
}

// The unique indexes in migration 00048 are the reason the planner exists. A
// duplicate here is a failed seed against Postgres, so it is caught in advance.
func TestUCGTaxonomyPlanSatisfiesUniqueIndexes(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	ids := map[uuid.UUID]string{}
	slugs := map[string]bool{}
	normalized := map[string]string{}
	for _, disease := range plan.Diseases {
		if previous, seen := ids[disease.ID]; seen {
			t.Errorf("duplicate disease id for %q and %q", previous, disease.Slug)
		}
		ids[disease.ID] = disease.Slug

		// idx_diseases_slug is unique on lower(slug).
		if slugs[strings.ToLower(disease.Slug)] {
			t.Errorf("duplicate slug %q", disease.Slug)
		}
		slugs[strings.ToLower(disease.Slug)] = true

		// idx_diseases_active_normalized_name is unique for active rows.
		if previous, seen := normalized[disease.NormalizedName]; seen {
			t.Errorf("duplicate normalized name %q for %q and %q", disease.NormalizedName, previous, disease.Slug)
		}
		normalized[disease.NormalizedName] = disease.Slug
	}

	// idx_disease_codes_system_code is unique across every disease.
	codes := map[string]string{}
	for _, code := range plan.Codes {
		key := strings.ToLower(code.CodeSystem) + "/" + strings.ToLower(code.Code)
		if previous, seen := codes[key]; seen {
			t.Errorf("ICD-10 %s claimed twice: %q then %q", code.Code, previous, ids[code.DiseaseID])
		}
		codes[key] = ids[code.DiseaseID]
	}

	// idx_disease_aliases_disease_normalized is unique per disease.
	aliases := map[string]bool{}
	for _, alias := range plan.Aliases {
		key := alias.DiseaseID.String() + "/" + alias.NormalizedAlias
		if aliases[key] {
			t.Errorf("duplicate alias %q on %q", alias.Alias, ids[alias.DiseaseID])
		}
		aliases[key] = true
	}
}

// trg_reject_disease_hierarchy_cycle rejects a parent that does not exist yet,
// so the write order has to put every parent before its children.
func TestUCGTaxonomyPlanWritesParentsFirst(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	known := map[uuid.UUID]bool{}
	for _, id := range ucgCuratedDiseaseIDs {
		known[id] = true
	}
	for _, disease := range plan.Diseases {
		if disease.ParentID != nil && !known[*disease.ParentID] {
			t.Errorf("%q is written before its parent", disease.Slug)
		}
		known[disease.ID] = true
	}

	for _, alias := range plan.Aliases {
		if !known[alias.DiseaseID] {
			t.Errorf("alias %q points at a disease that is never written", alias.Alias)
		}
	}
	for _, code := range plan.Codes {
		if !known[code.DiseaseID] {
			t.Errorf("code %s points at a disease that is never written", code.Code)
		}
	}
}

// Migration 00048 owns these rows. Reinserting them would break the unique
// index on an active normalized name, and overwriting them would discard the
// curated description, icon and colour.
func TestUCGTaxonomyReusesCuratedDiseases(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	bySlug := map[string]ucgPlannedDisease{}
	for _, disease := range plan.Diseases {
		bySlug[disease.Slug] = disease
	}

	for slug, id := range ucgCuratedDiseaseIDs {
		disease, ok := bySlug[slug]
		if !ok {
			t.Errorf("curated disease %q is missing from the plan", slug)
			continue
		}
		if !disease.Curated {
			t.Errorf("%q must be marked curated so its row is left alone", slug)
		}
		if disease.ID != id {
			t.Errorf("%q id = %s, want the existing %s", slug, disease.ID, id)
		}
	}

	// Malaria carries UCG children, which is the reason the curated id has to be
	// resolved rather than a fresh one minted.
	found := false
	for _, disease := range plan.Diseases {
		if disease.Slug == "uncomplicated-malaria" {
			found = true
			if disease.ParentID == nil || *disease.ParentID != ucgCuratedDiseaseIDs["malaria"] {
				t.Errorf("uncomplicated-malaria parent = %v, want the curated malaria id", disease.ParentID)
			}
		}
	}
	if !found {
		t.Error("uncomplicated-malaria is missing from the plan")
	}

	// A curated disease keeps whatever label its existing code rows carry, so
	// the seed sends no display name for them.
	curatedIDs := map[uuid.UUID]bool{}
	for _, id := range ucgCuratedDiseaseIDs {
		curatedIDs[id] = true
	}
	labelled := 0
	for _, code := range plan.Codes {
		if curatedIDs[code.DiseaseID] {
			if code.DisplayName != "" {
				t.Errorf("code %s is on a curated disease and must not set a display name", code.Code)
			}
			continue
		}
		if code.DisplayName == "" {
			t.Errorf("code %s should carry the guideline heading as its display name", code.Code)
		}
		labelled++
	}
	if labelled == 0 {
		t.Error("expected most codes to carry a display name")
	}
}

// normalizeUCGTerm has to agree with normalizeDiseaseTerm in the service, or
// lookups through the API will not find these rows. The fixture slug is derived
// from the same normalized string, so it doubles as the expected value.
func TestUCGTaxonomyNormalizationMatchesSlugs(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	for _, disease := range plan.Diseases {
		if want := strings.ReplaceAll(disease.NormalizedName, " ", "-"); want != disease.Slug {
			t.Errorf("%q normalizes to %q, which does not match its slug", disease.Name, disease.NormalizedName)
		}
	}

	for _, tc := range []struct{ in, want string }{
		{"COVID-19 Disease", "covid 19 disease"},
		{"Leprosy/Hansen’s Disease", "leprosy hansen s disease"},
		{"  Spaced   Out  ", "spaced out"},
	} {
		if got := normalizeUCGTerm(tc.in); got != tc.want {
			t.Errorf("normalizeUCGTerm(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Each of these is a decision the planner made on the reviewer's behalf, so it
// is pinned: the run log and the review document have to keep agreeing.
func TestUCGTaxonomyReconciliations(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	bySlug := map[string]ucgPlannedDisease{}
	for _, disease := range plan.Diseases {
		bySlug[disease.Slug] = disease
	}

	// The UCG lists these subjects under two chapters; the later one merges in.
	for _, slug := range []string{"cryptococcal-meningitis", "pelvic-inflammatory-disease", "condom"} {
		if _, ok := bySlug[slug]; !ok {
			t.Errorf("%q is missing from the plan", slug)
		}
	}

	// A struck parent leaves its clinical children behind, so they become roots
	// rather than disappearing with it.
	for _, slug := range []string{"drug-resistant-tb", "tuberculosis-in-children-and-adolescents", "post-tb-patient-management"} {
		disease, ok := bySlug[slug]
		if !ok {
			t.Fatalf("%q is missing from the plan", slug)
		}
		if disease.ParentID != nil {
			t.Errorf("%q should be re-parented to the root: its UCG parent was struck in review", slug)
		}
	}

	// Nothing struck in review may be seeded.
	for _, slug := range []string{"management-of-tb", "national-immunization-schedule", "monitoring-of-art"} {
		if _, ok := bySlug[slug]; ok {
			t.Errorf("%q is marked skip in the fixture but was planned", slug)
		}
	}

	byCode := map[string]string{}
	for _, code := range plan.Codes {
		byCode[code.Code] = code.DiseaseID.String()
	}
	// The first claimant of a repeated code keeps it.
	if byCode["S00-T88"] != bySlug["fractures"].ID.String() {
		t.Error("ICD-10 S00-T88 should stay with fractures, the first claimant")
	}
	if _, claimed := byCode["Z30.41"]; !claimed {
		t.Error("ICD-10 Z30.41 should be claimed once")
	}
}

// A rerun must produce the same rows or the derived identifiers stop being
// idempotent. Map iteration order is the usual way that breaks.
func TestUCGTaxonomyPlanIsDeterministic(t *testing.T) {
	first, rows := ucgTestPlan(t)
	second, err := planUCGDiseaseTaxonomy(rows)
	if err != nil {
		t.Fatalf("replan: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("planning the same rows twice produced different output")
	}
}

func TestUCGTaxonomyRejectsUnknownSeedAction(t *testing.T) {
	const fixture = "chapter,chapter_title,section_number,depth,name,short_name,aliases,slug,parent_name,icd10_codes,seed_action\r\n" +
		"1,Emergencies and Trauma,1.1.1,3,Anaphylactic Shock,,,anaphylactic-shock,,T78.2,maybe\r\n"
	if _, err := parseUCGTaxonomyRows(fixture); err == nil {
		t.Error("expected an error for an unrecognised seed_action")
	}
}

func TestUCGTaxonomyRejectsMissingColumn(t *testing.T) {
	const fixture = "chapter,section_number,depth,name,slug\r\n1,1.1.1,3,Anaphylactic Shock,anaphylactic-shock\r\n"
	if _, err := parseUCGTaxonomyRows(fixture); err == nil {
		t.Error("expected an error for a fixture missing required columns")
	}
}

// Filtering by category only works if nothing is left out of one, so every
// planned disease must be filed under a planned category.
// ucgSlugPattern mirrors the slug check on the diseases table, which is the
// stricter of the two tables a UCG slug lands in.
var ucgSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func TestUCGTaxonomyFilesEveryDiseaseUnderAChapter(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	categorySlugs := map[string]bool{}
	for index, category := range plan.Categories {
		if categorySlugs[category.Slug] {
			t.Errorf("category slug %q is planned twice", category.Slug)
		}
		categorySlugs[category.Slug] = true
		if !ucgSlugPattern.MatchString(category.Slug) {
			t.Errorf("category slug %q does not satisfy the slug pattern", category.Slug)
		}
		if want := (index + 1) * 10; category.SortOrder != want {
			t.Errorf("category %q sort order = %d, want %d", category.Slug, category.SortOrder, want)
		}
	}

	filed := map[uuid.UUID]bool{}
	seen := map[string]bool{}
	for _, link := range plan.CategoryLinks {
		if !categorySlugs[link.CategorySlug] {
			t.Errorf("disease %s is filed under unplanned category %q", link.DiseaseID, link.CategorySlug)
		}
		key := link.DiseaseID.String() + "/" + link.CategorySlug
		if seen[key] {
			t.Errorf("disease %s is filed under %q twice", link.DiseaseID, link.CategorySlug)
		}
		seen[key] = true
		filed[link.DiseaseID] = true
	}
	for _, disease := range plan.Diseases {
		if !filed[disease.ID] {
			t.Errorf("disease %q is not filed under any category", disease.Slug)
		}
	}
}

func TestUCGTaxonomyCategoriesFollowChapterTitles(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	for _, want := range []ucgPlannedCategory{
		{Name: "Emergencies and Trauma", Slug: "emergencies-and-trauma", SortOrder: 10},
		{Name: "Infectious Diseases", Slug: "infectious-diseases", SortOrder: 20},
		{Name: "Family Planning (FP)", Slug: "family-planning-fp", SortOrder: 150},
		{Name: "Ear, Nose & Throat Conditions", Slug: "ear-nose-throat-conditions", SortOrder: 210},
	} {
		found := false
		for _, category := range plan.Categories {
			if category.Slug != want.Slug {
				continue
			}
			found = true
			if category.Name != want.Name || category.SortOrder != want.SortOrder {
				t.Errorf("category %q = %q at %d, want %q at %d",
					want.Slug, category.Name, category.SortOrder, want.Name, want.SortOrder)
			}
			if category.ID != ucgTaxonomyID("category", want.Slug) {
				t.Errorf("category %q has an unstable id", want.Slug)
			}
		}
		if !found {
			t.Errorf("category %q is not planned", want.Slug)
		}
	}
}

// A subject the guideline repeats under a second chapter merges into the first
// row, but it must still be found when filtering by either chapter.
func TestUCGTaxonomyFilesMergedSubjectsUnderBothChapters(t *testing.T) {
	plan, _ := ucgTestPlan(t)

	categoriesOf := map[uuid.UUID][]string{}
	for _, link := range plan.CategoryLinks {
		categoriesOf[link.DiseaseID] = append(categoriesOf[link.DiseaseID], link.CategorySlug)
	}
	got := categoriesOf[ucgTaxonomyID("disease", "pelvic-inflammatory-disease")]
	want := []string{"hiv-aids-and-sexually-transmitted-infections", "gynaecological-conditions"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pelvic inflammatory disease is filed under %v, want %v", got, want)
	}

	// A curated disease is filed under its chapter like any other.
	got = categoriesOf[ucgCuratedDiseaseIDs["malaria"]]
	if !reflect.DeepEqual(got, []string{"infectious-diseases"}) {
		t.Errorf("malaria is filed under %v, want [infectious-diseases]", got)
	}
}

func TestUCGTaxonomyRejectsConflictingChapterTitles(t *testing.T) {
	header := "chapter,chapter_title,section_number,depth,name,short_name,aliases,slug,parent_name,icd10_codes,seed_action\n"
	rows, err := parseUCGTaxonomyRows(header +
		"1,Emergencies,1.1,2,Shock,,,shock,,,seed\n" +
		"1,Trauma,1.2,2,Burns,,,burns,,,seed\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := planUCGDiseaseTaxonomy(rows); err == nil || !strings.Contains(err.Error(), "titled both") {
		t.Errorf("err = %v, want a conflicting chapter title error", err)
	}
}
