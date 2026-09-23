package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mediguide/internal/middleware"
	"mediguide/internal/models"
	"mediguide/internal/security"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGuidelineContentWriteRequiresPermission(t *testing.T) {
	handler := testGuidelineContentHandler(t)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.New(), Perms: []string{"guideline.read"}})
		c.Next()
	})
	router.POST("/api/v2/guideline-categories", middleware.RequirePermission("guideline.write"), handler.CreateCategory)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/guideline-categories", strings.NewReader(`{"name":"Emergency"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestGuidelineContentWriterCanCreateCategory(t *testing.T) {
	handler := testGuidelineContentHandler(t)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.New(), Perms: []string{"guideline.write"}})
		c.Next()
	})
	router.POST("/api/v2/guideline-categories", middleware.RequirePermission("guideline.write"), handler.CreateCategory)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/guideline-categories", strings.NewReader(`{"name":"Emergency"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
}

func TestDocumentKindRoutes(t *testing.T) {
	handler := testGuidelineContentHandler(t)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.New(), Perms: []string{"guideline.write"}})
		c.Next()
	})
	router.POST("/api/v2/document-kinds", middleware.RequirePermission("guideline.write"), handler.CreateDocumentKind)
	router.DELETE("/api/v2/document-kinds/:id", middleware.RequirePermission("guideline.write"), handler.DeleteDocumentKind)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/document-kinds", strings.NewReader(`{"name":"Form"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"slug":"form"`) {
		t.Fatalf("expected 201 with slug, got %d: %s", response.Code, response.Body.String())
	}

	var kind models.DocumentKind
	if err := handler.Service.DB.First(&kind, "slug = ?", "form").Error; err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v2/document-kinds/"+kind.ID.String(), nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("deleting the only active kind: expected 409, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRejectPublishedAsUploadedGuardsEditRoutes(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.DocumentKind{}, &models.GuidelineDocument{}, &models.GuidelineVersion{}); err != nil {
		t.Fatal(err)
	}
	form := models.DocumentKind{Name: "Form", Slug: "form", Status: "active", PublishAsUploaded: true}
	guideline := models.DocumentKind{Name: "Guideline", Slug: "guideline", Status: "active"}
	for _, kind := range []*models.DocumentKind{&form, &guideline} {
		if err := database.Create(kind).Error; err != nil {
			t.Fatal(err)
		}
	}
	versions := map[string]uuid.UUID{}
	for name, kindID := range map[string]uuid.UUID{"form": form.ID, "guideline": guideline.ID} {
		document := models.GuidelineDocument{Title: name, DocumentKindID: &kindID}
		if err := database.Create(&document).Error; err != nil {
			t.Fatal(err)
		}
		version := models.GuidelineVersion{DocumentID: document.ID, Version: "1", Status: "draft"}
		if err := database.Create(&version).Error; err != nil {
			t.Fatal(err)
		}
		versions[name] = version.ID
	}
	handler := GuidelineHandler{Service: services.GuidelineService{DB: database}}
	router := gin.New()
	router.PUT("/api/v2/guideline-versions/:id/markdown-draft", handler.RejectPublishedAsUploaded, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for name, want := range map[string]int{"form": http.StatusConflict, "guideline": http.StatusNoContent} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v2/guideline-versions/"+versions[name].String()+"/markdown-draft", nil))
		if response.Code != want {
			t.Fatalf("%s version: expected %d, got %d: %s", name, want, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v2/guideline-versions/"+uuid.NewString()+"/markdown-draft", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown version: expected 404, got %d", response.Code)
	}
}

func testGuidelineContentHandler(t *testing.T) GuidelineContentHandler {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.GuidelineCategory{}, &models.GuidelineTag{}, &models.Abbreviation{}, &models.GuidelineIndexEntry{}, &models.MedicalGuideline{}, &models.DocumentKind{}, &models.GuidelineDocument{}, &models.OutbreakResource{}); err != nil {
		t.Fatal(err)
	}
	return GuidelineContentHandler{Service: services.GuidelineContentService{DB: database}}
}
