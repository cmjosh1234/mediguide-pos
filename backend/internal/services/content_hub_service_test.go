package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func contentHubTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := classificationTestDB(t)
	if err := db.AutoMigrate(
		&models.DiseaseAlias{},
		&models.DiseaseCode{},
		&models.ContentHub{},
		&models.ContentHubDisease{},
		&models.ContentHubOutbreak{},
		&models.ContentPillar{},
		&models.ContentPillarItem{},
		&models.ContentHubTemplate{},
		&models.ContentHubTemplatePillar{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestConfigureOutbreakHubMapsPublishedResourcesAndSurveillance(t *testing.T) {
	db := contentHubTestDB(t)
	template := models.ContentHubTemplate{Base: models.Base{ID: defaultOutbreakHubTemplateID}, Name: "Outbreak response", Slug: "outbreak-emergency-response", Status: models.ContentHubStatusActive}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	definitions := []models.ContentHubTemplatePillar{
		{TemplateID: template.ID, Name: "Surveillance Guidance", Slug: "surveillance-guidance", SortOrder: 10},
		{TemplateID: template.ID, Name: "Clinical Management", Slug: "clinical-management", SortOrder: 20},
		{TemplateID: template.ID, Name: "Situation Reports", Slug: "situation-reports", SortOrder: 30},
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	outbreak := models.Outbreak{Title: "Ebola", Status: "active", LastUpdate: now, PublishedAt: &now}
	if err := db.Create(&outbreak).Error; err != nil {
		t.Fatal(err)
	}
	typedResource := models.OutbreakResource{OutbreakID: outbreak.ID, Title: "Surveillance protocol", ResourceType: "guideline", DocumentKind: "surveillance_protocol", Status: "published", PublishedAt: &now, ApprovedAt: &now}
	if err := db.Create(&typedResource).Error; err != nil {
		t.Fatal(err)
	}
	report := models.SituationReport{OutbreakID: &outbreak.ID, Title: "Situation report", Status: "published", PublicationDate: now, PublishedAt: &now, ApprovedAt: &now}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	service := ContentHubService{DB: db}
	workspace, err := service.ConfigureOutbreakHub(ContentHubActor{}, outbreak.ID, ConfigureOutbreakHubInput{Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Hub.Status != models.ContentHubStatusActive || len(workspace.Hub.Outbreaks) != 1 {
		t.Fatalf("outbreak hub not published or linked: %#v", workspace.Hub)
	}
	bySlug := map[string]ContentHubAdminPillar{}
	for _, pillar := range workspace.Pillars {
		bySlug[pillar.Slug] = pillar
	}
	if got := bySlug["surveillance-guidance"].Items; len(got) != 0 {
		t.Fatalf("typed resources must not be mapped into hub pillars: %#v", got)
	}
	if got := bySlug["situation-reports"].Items; len(got) != 1 || got[0].ContentID == nil || *got[0].ContentID != report.ID {
		t.Fatalf("report mapping failed: %#v", got)
	}
	publicHub, err := service.GetPublicOutbreakHub(context.Background(), outbreak.ID)
	if err != nil || publicHub.OutbreakID == nil || *publicHub.OutbreakID != outbreak.ID {
		t.Fatalf("public outbreak lookup failed: %#v %v", publicHub, err)
	}
	publicBySlug, err := service.GetPublicHub(context.Background(), workspace.Hub.Slug)
	if err != nil || publicBySlug.Outbreak == nil || publicBySlug.Outbreak.ID != outbreak.ID {
		t.Fatalf("normal public hub route omitted its outbreak banner: %#v %v", publicBySlug, err)
	}
	again, err := service.ConfigureOutbreakHub(ContentHubActor{}, outbreak.ID, ConfigureOutbreakHubInput{})
	if err != nil || again.Hub.ID != workspace.Hub.ID {
		t.Fatalf("configuration should be idempotent: %#v %v", again, err)
	}
}

func TestConfigureOutbreakHubCarriesOverTheOutbreaksDisease(t *testing.T) {
	db := contentHubTestDB(t)
	template := models.ContentHubTemplate{Base: models.Base{ID: defaultOutbreakHubTemplateID}, Name: "Outbreak response", Slug: "outbreak-emergency-response", Status: models.ContentHubStatusActive}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContentHubTemplatePillar{TemplateID: template.ID, Name: "Overview", Slug: "overview", SortOrder: 10}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	cholera := models.Disease{Name: "Cholera", Slug: "cholera", NormalizedName: "cholera", Status: models.DiseaseStatusActive}
	archived := models.Disease{Name: "Old condition", Slug: "old-condition", NormalizedName: "old condition", Status: models.DiseaseStatusArchived}
	if err := db.Create(&cholera).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&archived).Error; err != nil {
		t.Fatal(err)
	}
	service := ContentHubService{DB: db}

	withDisease := models.Outbreak{Title: "Cholera response", Status: "active", LastUpdate: now, PublishedAt: &now, DiseaseID: &cholera.ID}
	if err := db.Create(&withDisease).Error; err != nil {
		t.Fatal(err)
	}
	workspace, err := service.ConfigureOutbreakHub(ContentHubActor{}, withDisease.ID, ConfigureOutbreakHubInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Hub.Diseases) != 1 || workspace.Hub.Diseases[0].ID != cholera.ID {
		t.Fatalf("expected the outbreak's disease to carry over: hub=%#v", workspace.Hub)
	}

	withoutDisease := models.Outbreak{Title: "Unlinked response", Status: "active", LastUpdate: now, PublishedAt: &now}
	if err := db.Create(&withoutDisease).Error; err != nil {
		t.Fatal(err)
	}
	noDiseaseWorkspace, err := service.ConfigureOutbreakHub(ContentHubActor{}, withoutDisease.ID, ConfigureOutbreakHubInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(noDiseaseWorkspace.Hub.Diseases) != 0 {
		t.Fatalf("outbreak without a disease should produce a hub without one: hub=%#v", noDiseaseWorkspace.Hub)
	}

	withArchivedDisease := models.Outbreak{Title: "Stale disease response", Status: "active", LastUpdate: now, PublishedAt: &now, DiseaseID: &archived.ID}
	if err := db.Create(&withArchivedDisease).Error; err != nil {
		t.Fatal(err)
	}
	archivedWorkspace, err := service.ConfigureOutbreakHub(ContentHubActor{}, withArchivedDisease.ID, ConfigureOutbreakHubInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(archivedWorkspace.Hub.Diseases) != 0 {
		t.Fatalf("an archived disease should be skipped, not fail hub creation: hub=%#v", archivedWorkspace.Hub)
	}
}

func TestContentHubDiseaseReplacementUsesLockVersionAndPreviewEligibility(t *testing.T) {
	db := contentHubTestDB(t)
	disease := models.Disease{Name: "Malaria", Slug: "malaria", NormalizedName: "malaria", Status: models.DiseaseStatusActive}
	if err := db.Create(&disease).Error; err != nil {
		t.Fatal(err)
	}
	service := ContentHubService{DB: db}
	hub, err := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Malaria hub"})
	if err != nil {
		t.Fatal(err)
	}
	hub, err = service.ReplaceHubDiseases(ContentHubActor{}, hub.ID, ReplaceContentHubDiseasesInput{DiseaseIDs: []uuid.UUID{disease.ID}, LockVersion: hub.LockVersion})
	if err != nil || len(hub.Diseases) != 1 {
		t.Fatalf("disease relationship was not replaced: %#v %v", hub, err)
	}
	if _, err := service.ReplaceHubDiseases(ContentHubActor{}, hub.ID, ReplaceContentHubDiseasesInput{DiseaseIDs: nil, LockVersion: 1}); !errors.Is(err, ErrContentHubConflict) {
		t.Fatalf("expected stale relationship update conflict, got %v", err)
	}
	preview, err := service.PreviewHub(context.Background(), hub.ID)
	if err != nil || len(preview.Diseases) != 1 || preview.Diseases[0].ID != disease.ID {
		t.Fatalf("draft preview did not use public projection: %#v %v", preview, err)
	}
}

func TestPublicDiseaseDirectoryKeepsParentsWithEligibleDescendants(t *testing.T) {
	db := contentHubTestDB(t)
	parent := models.Disease{Name: "Communicable infection", Slug: "communicable-infection", NormalizedName: "communicable infection", Status: models.DiseaseStatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Disease{Name: "Ebola virus disease", Slug: "ebola-virus-disease", NormalizedName: "ebola virus disease", ParentID: &parent.ID, Status: models.DiseaseStatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	alias := models.DiseaseAlias{DiseaseID: child.ID, Alias: "EVD", NormalizedAlias: "evd"}
	if err := db.Create(&alias).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	hub := models.ContentHub{Name: "Ebola care", Slug: "ebola-care", Status: models.ContentHubStatusActive, PublishedAt: &now}
	if err := db.Create(&hub).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContentHubDisease{ContentHubID: hub.ID, DiseaseID: child.ID}).Error; err != nil {
		t.Fatal(err)
	}
	service := DiseaseService{DB: db}
	page, err := service.ListPublic(context.Background(), PublicDiseaseQuery{Page: PageInput{Page: 1, PerPage: 20}})
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != parent.ID || page.Items[1].ID != child.ID {
		t.Fatalf("eligible hierarchy was flattened or parent hidden: %#v %v", page, err)
	}
	tree, err := service.PublicHierarchy(context.Background())
	if err != nil || len(tree) != 1 || tree[0].ID != parent.ID || len(tree[0].Children) != 1 || tree[0].Children[0].ID != child.ID {
		t.Fatalf("eligible public hierarchy is incorrect: %#v %v", tree, err)
	}
	resolved, err := service.GetPublic(context.Background(), "EVD")
	if err != nil || resolved.ID != child.ID {
		t.Fatalf("public alias did not resolve to the canonical disease: %#v %v", resolved, err)
	}
}

func TestContentHubSupportsNeutralAndMultipleDiseaseHubs(t *testing.T) {
	db := contentHubTestDB(t)
	malaria := models.Disease{Name: "Malaria", Slug: "malaria", NormalizedName: "malaria", Status: models.DiseaseStatusActive}
	ebola := models.Disease{Name: "Ebola virus disease", Slug: "ebola", NormalizedName: "ebola virus disease", Status: models.DiseaseStatusActive}
	archived := models.Disease{Name: "Old condition", Slug: "old-condition", NormalizedName: "old condition", Status: models.DiseaseStatusArchived}
	for _, disease := range []*models.Disease{&malaria, &ebola, &archived} {
		if err := db.Create(disease).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := ContentHubService{DB: db}
	neutral, err := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "General clinical knowledge", Slug: "general-clinical-knowledge"})
	if err != nil || len(neutral.Diseases) != 0 {
		t.Fatalf("neutral hub failed: hub=%#v err=%v", neutral, err)
	}
	multi, err := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Febrile illness", Slug: "febrile-illness", DiseaseIDs: []uuid.UUID{malaria.ID, ebola.ID}})
	if err != nil || len(multi.Diseases) != 2 {
		t.Fatalf("multi-disease hub failed: hub=%#v err=%v", multi, err)
	}
	secondMalariaHub, err := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Malaria training", Slug: "malaria-training", DiseaseIDs: []uuid.UUID{malaria.ID}})
	if err != nil {
		t.Fatalf("a disease should support multiple hubs: %v", err)
	}
	page, err := service.ListHubs(ContentHubQuery{Page: PageInput{Page: 1, PerPage: 20}, DiseaseID: malaria.ID.String()})
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != multi.ID || page.Items[1].ID != secondMalariaHub.ID {
		t.Fatalf("disease filtering failed: page=%#v err=%v", page, err)
	}
	if _, err := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Invalid", Slug: "invalid", DiseaseIDs: []uuid.UUID{archived.ID}}); !errors.Is(err, ErrContentDiseaseUnavailable) {
		t.Fatalf("archived disease should be rejected, got %v", err)
	}
}

func TestContentPillarHierarchyRejectsCyclesAndCrossHubParents(t *testing.T) {
	service := ContentHubService{DB: contentHubTestDB(t)}
	first, _ := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "First hub", Slug: "first-hub"})
	second, _ := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Second hub", Slug: "second-hub"})
	parent, err := service.CreatePillar(ContentHubActor{}, first.ID, ContentPillarInput{Name: "Clinical care", Slug: "clinical-care"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.CreatePillar(ContentHubActor{}, first.ID, ContentPillarInput{Name: "Diagnosis", Slug: "diagnosis", ParentID: &parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdatePillar(ContentHubActor{}, first.ID, parent.ID, ContentPillarInput{Name: parent.Name, Slug: parent.Slug, ParentID: &child.ID, LockVersion: parent.LockVersion}); !errors.Is(err, ErrContentPillarCycle) {
		t.Fatalf("cycle should be rejected, got %v", err)
	}
	if _, err := service.CreatePillar(ContentHubActor{}, second.ID, ContentPillarInput{Name: "Wrong parent", Slug: "wrong-parent", ParentID: &parent.ID}); !errors.Is(err, ErrContentPillarWrongHub) {
		t.Fatalf("cross-hub parent should be rejected, got %v", err)
	}
}

func TestHubTemplateIsCopiedOnceAndRemainsEditable(t *testing.T) {
	db := contentHubTestDB(t)
	template := models.ContentHubTemplate{Name: "Disease care", Slug: "disease-care", Status: models.ContentHubStatusActive}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	definitions := []models.ContentHubTemplatePillar{
		{TemplateID: template.ID, Name: "Overview", Slug: "overview", SortOrder: 10},
		{TemplateID: template.ID, Name: "Diagnosis", Slug: "diagnosis", SortOrder: 20},
		{TemplateID: template.ID, Name: "Clinical Management", Slug: "clinical-management", SortOrder: 30},
		{TemplateID: template.ID, Name: "Medicines", Slug: "medicines", SortOrder: 40},
		{TemplateID: template.ID, Name: "Algorithms", Slug: "algorithms", SortOrder: 50},
		{TemplateID: template.ID, Name: "Prevention", Slug: "prevention", SortOrder: 60},
		{TemplateID: template.ID, Name: "Patient Education", Slug: "patient-education", SortOrder: 70},
		{TemplateID: template.ID, Name: "Training", Slug: "training", SortOrder: 80},
		{TemplateID: template.ID, Name: "References", Slug: "references", SortOrder: 90},
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatal(err)
	}
	service := ContentHubService{DB: db}
	hub, _ := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Diabetes care", Slug: "diabetes-care"})
	pillars, err := service.ApplyTemplate(ContentHubActor{}, hub.ID, ApplyContentHubTemplateInput{TemplateID: template.ID, LockVersion: hub.LockVersion})
	if err != nil || len(pillars) != 9 || pillars[0].Slug != "overview" || pillars[8].Slug != "references" {
		t.Fatalf("template copy failed: pillars=%#v err=%v", pillars, err)
	}
	updated, err := service.UpdatePillar(ContentHubActor{}, hub.ID, pillars[0].ID, ContentPillarInput{Name: "At a glance", Slug: "at-a-glance", SortOrder: 5, LockVersion: pillars[0].LockVersion})
	if err != nil || updated.Name != "At a glance" {
		t.Fatalf("copied pillar was not editable: %#v %v", updated, err)
	}
	reloaded, _ := service.GetHub(hub.ID)
	if _, err := service.ApplyTemplate(ContentHubActor{}, hub.ID, ApplyContentHubTemplateInput{TemplateID: template.ID, LockVersion: reloaded.LockVersion}); !errors.Is(err, ErrContentHubNotEmpty) {
		t.Fatalf("second template application should be rejected, got %v", err)
	}
	storedTemplate, _ := service.GetTemplate(template.ID)
	if storedTemplate.Pillars[0].Name != "Overview" {
		t.Fatal("editing a copied pillar mutated the template")
	}
}

func TestContentHubPublicEligibilityAndNonDestructiveItemRemoval(t *testing.T) {
	db := contentHubTestDB(t)
	service := ContentHubService{DB: db, AllowedExternalHosts: []string{"who.int"}}
	publishedDocument := models.GuidelineDocument{Title: "Approved guidance"}
	draftDocument := models.GuidelineDocument{Title: "Draft guidance"}
	for _, document := range []*models.GuidelineDocument{&publishedDocument, &draftDocument} {
		if err := db.Create(document).Error; err != nil {
			t.Fatal(err)
		}
	}
	version := models.GuidelineVersion{DocumentID: publishedDocument.ID, Version: "1", Status: "published"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&publishedDocument).Update("current_version_id", version.ID).Error; err != nil {
		t.Fatal(err)
	}
	hub, _ := service.CreateHub(ContentHubActor{}, CreateContentHubInput{Name: "Care hub", Slug: "care-hub"})
	pillar, _ := service.CreatePillar(ContentHubActor{}, hub.ID, ContentPillarInput{Name: "Guidelines", Slug: "guidelines"})
	publishedItem, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentDiseaseGuideline, ContentID: &publishedDocument.ID, Status: models.ContentPillarItemStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentDiseaseGuideline, ContentID: &publishedDocument.ID, Status: models.ContentPillarItemStatusActive}); !errors.Is(err, ErrContentHubDuplicate) {
		t.Fatalf("duplicate pillar item should be rejected, got %v", err)
	}
	if _, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentDiseaseGuideline, ContentID: &draftDocument.ID, SortOrder: 20, Status: models.ContentPillarItemStatusActive}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Hour)
	if _, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentPillarItemApprovedExternalURL, Target: "https://who.int/resource", EndsAt: &past, SortOrder: 30, Status: models.ContentPillarItemStatusActive}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentPillarItemApprovedExternalURL, Target: "https://evil.example/resource", Status: models.ContentPillarItemStatusActive}); !errors.Is(err, ErrContentPillarUnsafeTarget) {
		t.Fatalf("non-allowlisted URL should be rejected, got %v", err)
	}
	if _, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentPillarItemInternalRoute, Target: "/public/guidelines/" + publishedDocument.ID.String(), SortOrder: 40, Status: models.ContentPillarItemStatusActive}); err != nil {
		t.Fatalf("approved internal route failed: %v", err)
	}
	draftItem, err := service.CreatePillarItem(ContentHubActor{}, hub.ID, pillar.ID, ContentPillarItemInput{ContentType: models.ContentPillarItemApprovedExternalURL, Target: "https://who.int/draft", SortOrder: 50})
	if err != nil || draftItem.Status != models.ContentPillarItemStatusDraft {
		t.Fatalf("new pillar items must default to draft: item=%#v err=%v", draftItem, err)
	}
	if _, err := service.GetPublicHub(context.Background(), hub.Slug); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("draft hub leaked publicly: %v", err)
	}
	hub, err = service.TransitionHub(ContentHubActor{}, hub.ID, "publish", ContentHubTransitionInput{LockVersion: hub.LockVersion})
	if err != nil {
		t.Fatal(err)
	}
	publicHub, err := service.GetPublicHub(context.Background(), hub.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(publicHub.Pillars) != 1 || len(publicHub.Pillars[0].Items) != 2 {
		t.Fatalf("public hub should keep eligible resource and route, skipping draft/expired items: %#v", publicHub.Pillars)
	}
	if err := service.DeletePillarItem(ContentHubActor{}, hub.ID, pillar.ID, publishedItem.ID, publishedItem.LockVersion); err != nil {
		t.Fatal(err)
	}
	var sourceCount int64
	if err := db.Model(&models.GuidelineDocument{}).Where("id = ?", publishedDocument.ID).Count(&sourceCount).Error; err != nil || sourceCount != 1 {
		t.Fatalf("removing a pillar assignment deleted the source: count=%d err=%v", sourceCount, err)
	}
}

func TestAssignableGuidelinesFilterByDocumentKind(t *testing.T) {
	db := contentHubTestDB(t)
	var sop, form models.DocumentKind
	if err := db.Where("slug = ?", "sop").First(&sop).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("slug = ?", "form").First(&form).Error; err != nil {
		t.Fatal(err)
	}
	documents := []models.GuidelineDocument{
		{Title: "Hand hygiene SOP", DocumentKindID: &sop.ID},
		{Title: "Referral form", DocumentKindID: &form.ID},
	}
	if err := db.Create(&documents).Error; err != nil {
		t.Fatal(err)
	}
	service := ContentHubService{DB: db}

	all, err := service.SearchAssignableResources(ContentHubResourceQuery{ContentType: "guideline"})
	if err != nil || len(all.Items) != 2 {
		t.Fatalf("expected every kind without a filter: items=%+v err=%v", all, err)
	}
	filtered, err := service.SearchAssignableResources(ContentHubResourceQuery{ContentType: "guideline", DocumentKindID: sop.ID.String()})
	if err != nil || len(filtered.Items) != 1 || filtered.Items[0].Title != "Hand hygiene SOP" {
		t.Fatalf("expected only the SOP: items=%+v err=%v", filtered, err)
	}
	if _, err := service.SearchAssignableResources(ContentHubResourceQuery{ContentType: "guideline", DocumentKindID: "not-a-uuid"}); !errors.Is(err, ErrContentHubInvalid) {
		t.Fatalf("expected an invalid kind to be rejected, got %v", err)
	}
}

func TestDeletePillarNeedsAReasonAndStaysInTheHubHistory(t *testing.T) {
	db := contentHubTestDB(t)
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "Hub Editor", Email: "editor@example.test", PasswordHash: "x", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	actor := ContentHubActor{ID: user.ID}
	service := ContentHubService{DB: db}
	hub, err := service.CreateHub(actor, CreateContentHubInput{Name: "Malaria", Audience: "all"})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := service.CreatePillar(actor, hub.ID, ContentPillarInput{Name: "Treatment"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePillar(actor, hub.ID, ContentPillarInput{Name: "Severe malaria", ParentID: &parent.ID}); err != nil {
		t.Fatal(err)
	}

	if err := service.DeletePillar(actor, hub.ID, parent.ID, parent.LockVersion, "  "); !errors.Is(err, ErrContentPillarNoReason) {
		t.Fatalf("expected a blank reason to be refused, got %v", err)
	}
	if err := service.DeletePillar(actor, hub.ID, parent.ID, parent.LockVersion, "Merged into Clinical Management"); err != nil {
		t.Fatal(err)
	}
	remaining, err := service.ListPillars(hub.ID)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("expected the pillar and its sub-pillar to be deleted: %+v err=%v", remaining, err)
	}

	history, err := service.ListAudit(hub.ID, PageInput{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	var deleted *ContentHubAuditRecord
	for index := range history.Items {
		if history.Items[index].Action == "content_pillar.delete" {
			deleted = &history.Items[index]
		}
	}
	if deleted == nil {
		t.Fatalf("the deletion is missing from the hub history: %+v", history.Items)
	}
	if deleted.Metadata["reason"] != "Merged into Clinical Management" || deleted.Metadata["name"] != "Treatment" || deleted.Metadata["removed_sub_pillars"] != float64(1) {
		t.Fatalf("unexpected deletion metadata: %+v", deleted.Metadata)
	}
	if deleted.ActorName != "Hub Editor" || deleted.ActorEmail != "editor@example.test" {
		t.Fatalf("expected the editor to be named, got %q %q", deleted.ActorName, deleted.ActorEmail)
	}
}

func TestUnarchivingAPillarIsRecordedAsAStatusChange(t *testing.T) {
	db := contentHubTestDB(t)
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	actor := ContentHubActor{ID: uuid.New()}
	service := ContentHubService{DB: db}
	hub, err := service.CreateHub(actor, CreateContentHubInput{Name: "Ebola", Audience: "all"})
	if err != nil {
		t.Fatal(err)
	}
	pillar, err := service.CreatePillar(actor, hub.ID, ContentPillarInput{Name: "IPC & PPE", Description: "Infection prevention.", SortOrder: 40})
	if err != nil {
		t.Fatal(err)
	}
	stored := ContentPillarInput{Name: pillar.Name, Slug: pillar.Slug, Description: pillar.Description, SortOrder: pillar.SortOrder}
	archive := stored
	archive.Status, archive.LockVersion = models.ContentPillarStatusArchived, pillar.LockVersion
	archived, err := service.UpdatePillar(actor, hub.ID, pillar.ID, archive)
	if err != nil {
		t.Fatal(err)
	}
	restore := stored
	restore.Status, restore.LockVersion = models.ContentPillarStatusActive, archived.LockVersion
	restored, err := service.UpdatePillar(actor, hub.ID, pillar.ID, restore)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != models.ContentPillarStatusActive || restored.Description != "Infection prevention." || restored.SortOrder != 40 {
		t.Fatalf("unarchiving changed more than the status: %+v", restored)
	}

	history, err := service.ListAudit(hub.ID, PageInput{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	moves := []string{}
	for _, entry := range history.Items {
		if entry.Action == "content_pillar.update" {
			moves = append(moves, fmt.Sprintf("%v->%v", entry.Metadata["from_status"], entry.Metadata["to_status"]))
		}
	}
	// Newest first.
	if strings.Join(moves, ",") != "archived->active,active->archived" {
		t.Fatalf("unexpected status history: %v", moves)
	}
}

func TestReplacingADraftHubsTemplateNeedsAReasonAndSwapsThePillars(t *testing.T) {
	db := contentHubTestDB(t)
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	newTemplate := func(name string, pillars ...string) models.ContentHubTemplate {
		template := models.ContentHubTemplate{Name: name, Slug: strings.ToLower(strings.ReplaceAll(name, " ", "-")), Status: models.ContentHubStatusActive}
		if err := db.Create(&template).Error; err != nil {
			t.Fatal(err)
		}
		for index, pillar := range pillars {
			definition := models.ContentHubTemplatePillar{TemplateID: template.ID, Name: pillar, Slug: strings.ToLower(pillar), SortOrder: (index + 1) * 10}
			if err := db.Create(&definition).Error; err != nil {
				t.Fatal(err)
			}
		}
		return template
	}
	care := newTemplate("Disease care", "Overview", "Diagnosis")
	surveillance := newTemplate("Surveillance knowledge", "Surveillance", "Reporting", "Overview")
	actor := ContentHubActor{ID: uuid.New()}
	service := ContentHubService{DB: db}
	hub, err := service.CreateHub(actor, CreateContentHubInput{Name: "Cholera", Audience: "all"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.ApplyTemplate(actor, hub.ID, ApplyContentHubTemplateInput{TemplateID: care.ID, LockVersion: hub.LockVersion})
	if err != nil || len(first) != 2 {
		t.Fatalf("first template not applied: %+v %v", first, err)
	}
	report := models.SituationReport{Title: "Week 1", Status: "published", PublicationDate: time.Now().UTC()}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContentPillarItem{PillarID: first[0].ID, ContentType: models.ContentDiseaseSituationReport, ContentID: &report.ID, Status: models.ContentPillarItemStatusActive, LockVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}

	reloaded, _ := service.GetHub(hub.ID)
	replace := ApplyContentHubTemplateInput{TemplateID: surveillance.ID, LockVersion: reloaded.LockVersion, Replace: true}
	if _, err := service.ApplyTemplate(actor, hub.ID, replace); !errors.Is(err, ErrContentHubReplaceNoReason) {
		t.Fatalf("expected a replacement without a reason to be refused, got %v", err)
	}
	replace.Reason = "Cholera is tracked as a surveillance hub"
	pillars, err := service.ApplyTemplate(actor, hub.ID, replace)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, pillar := range pillars {
		names = append(names, pillar.Name)
	}
	if strings.Join(names, ",") != "Surveillance,Reporting,Overview" {
		t.Fatalf("expected only the new template's pillars, got %v", names)
	}
	var liveItems int64
	if err := db.Model(&models.ContentPillarItem{}).Where("pillar_id = ?", first[0].ID).Count(&liveItems).Error; err != nil || liveItems != 0 {
		t.Fatalf("the old pillar's assignment should be deleted: count=%d err=%v", liveItems, err)
	}
	var keptReport int64
	if err := db.Model(&models.SituationReport{}).Where("id = ?", report.ID).Count(&keptReport).Error; err != nil || keptReport != 1 {
		t.Fatalf("the assigned report itself must be kept: count=%d err=%v", keptReport, err)
	}

	history, err := service.ListAudit(hub.ID, PageInput{Page: 1, PerPage: 50})
	if err != nil || len(history.Items) == 0 || history.Items[0].Action != "content_hub.apply_template" {
		t.Fatalf("expected the replacement to be the newest audit entry: %+v %v", history, err)
	}
	meta := history.Items[0].Metadata
	if meta["replace"] != true || meta["reason"] != replace.Reason || meta["template_name"] != "Surveillance knowledge" || meta["removed_pillars"] != float64(2) || meta["removed_assignments"] != float64(1) {
		t.Fatalf("unexpected replacement metadata: %+v", meta)
	}
}

func TestUnarchivingAHubReturnsItToDraft(t *testing.T) {
	db := contentHubTestDB(t)
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	actor := ContentHubActor{ID: uuid.New()}
	service := ContentHubService{DB: db}
	hub, err := service.CreateHub(actor, CreateContentHubInput{Name: "Malaria", Audience: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePillar(actor, hub.ID, ContentPillarInput{Name: "Overview"}); err != nil {
		t.Fatal(err)
	}
	step := func(action string) *models.ContentHub {
		t.Helper()
		current, err := service.GetHub(hub.ID)
		if err != nil {
			t.Fatal(err)
		}
		next, err := service.TransitionHub(actor, hub.ID, action, ContentHubTransitionInput{LockVersion: current.LockVersion})
		if err != nil {
			t.Fatalf("%s failed: %v", action, err)
		}
		return next
	}
	step("publish")
	current, _ := service.GetHub(hub.ID)
	if _, err := service.TransitionHub(actor, hub.ID, "unarchive", ContentHubTransitionInput{LockVersion: current.LockVersion}); !errors.Is(err, ErrContentHubInvalid) {
		t.Fatalf("only an archived hub can be unarchived, got %v", err)
	}
	step("archive")
	if restored := step("unarchive"); restored.Status != models.ContentHubStatusDraft {
		t.Fatalf("expected the hub back in draft, got %q", restored.Status)
	}
	if republished := step("publish"); republished.Status != models.ContentHubStatusActive {
		t.Fatalf("expected the unarchived hub to publish again, got %q", republished.Status)
	}

	history, err := service.ListAudit(hub.ID, PageInput{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	moves := []string{}
	for _, entry := range history.Items {
		if entry.EntityType == "content_hub" && entry.Metadata["to_status"] != nil {
			moves = append(moves, fmt.Sprintf("%s:%v->%v", entry.Action, entry.Metadata["from_status"], entry.Metadata["to_status"]))
		}
	}
	want := "content_hub.publish:draft->active,content_hub.unarchive:archived->draft,content_hub.archive:active->archived,content_hub.publish:draft->active"
	if strings.Join(moves, ",") != want {
		t.Fatalf("unexpected hub status history:\n got %s\nwant %s", strings.Join(moves, ","), want)
	}
}
