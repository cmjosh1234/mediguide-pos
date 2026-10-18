package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mediguide/internal/middleware"
	"mediguide/internal/models"
	"mediguide/internal/security"
	"mediguide/internal/services"
)

func TestGuidelineRevisionPermissionsAndValidation(t *testing.T) {
	db := newGuidelineLibraryHandlerDB(t)
	if err := db.AutoMigrate(&models.DocumentKind{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	kind := models.DocumentKind{Name: "Link", Slug: "link", PublishAsLink: true, Status: "active"}
	if err := db.Create(&kind).Error; err != nil {
		t.Fatal(err)
	}
	doc := createPublishedGuidelineLibraryHandlerDocument(t, db)
	source := *doc.CurrentVersionID
	now := time.Now().UTC()
	if err := db.Model(&doc).Updates(map[string]any{"current_version_id": nil, "withdrawn_version_id": source, "withdrawn_at": now, "document_kind_id": kind.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.GuidelineVersion{}).Where("id = ?", source).Update("external_url", "https://example.org/source").Error; err != nil {
		t.Fatal(err)
	}
	h := GuidelineHandler{Service: services.GuidelineService{DB: db}}
	body := `{"source_version_id":"` + source.String() + `","version":"2"}`
	for _, test := range []struct {
		name     string
		perms    []string
		id, body string
		status   int
	}{
		{"reader forbidden", []string{"guideline.read"}, doc.ID.String(), body, 403},
		{"writer without edit forbidden", []string{"guideline.write"}, doc.ID.String(), body, 403},
		{"editor without write forbidden", []string{"guideline.markdown.edit"}, doc.ID.String(), body, 403},
		{"bad id", []string{"guideline.write", "guideline.markdown.edit"}, "bad", body, 400},
		{"missing source", []string{"guideline.write", "guideline.markdown.edit"}, doc.ID.String(), `{"version":"2"}`, 400},
		{"stale source", []string{"guideline.write", "guideline.markdown.edit"}, doc.ID.String(), `{"version":"2","source_version_id":"` + uuid.NewString() + `"}`, 409},
		{"create draft", []string{"guideline.write", "guideline.markdown.edit"}, doc.ID.String(), body, 200},
		{"continue draft", []string{"guideline.write", "guideline.markdown.edit"}, doc.ID.String(), body, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.New(), Perms: test.perms})
				c.Next()
			})
			router.POST("/guidelines/:id/revise", middleware.RequirePermission("guideline.write"), middleware.RequirePermission("guideline.markdown.edit"), h.ReviseWithdrawnDocument)
			req := httptest.NewRequest(http.MethodPost, "/guidelines/"+test.id+"/revise", strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != test.status {
				t.Fatalf("got %d want %d: %s", res.Code, test.status, res.Body.String())
			}
		})
	}
}
