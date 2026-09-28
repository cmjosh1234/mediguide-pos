package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"mediguide/internal/models"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type outbreakDownloadHandlerStore struct{ objects map[string][]byte }

func (s outbreakDownloadHandlerStore) Put(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (s outbreakDownloadHandlerStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.objects[key])), nil
}
func (s outbreakDownloadHandlerStore) Delete(context.Context, string) error { return nil }
func (s outbreakDownloadHandlerStore) PresignGet(context.Context, string, time.Duration) (*url.URL, error) {
	return url.Parse("http://minio:9000/private-object")
}

func publicOutbreakTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Disease{}, &models.Outbreak{}, &models.OutbreakUpdate{}, &models.OutbreakResource{}, &models.SituationReport{}); err != nil {
		t.Fatal(err)
	}
	handler := OutbreakHandler{Service: services.OutbreakService{DB: db}}
	router := gin.New()
	router.GET("/api/public/outbreaks", handler.List)
	router.GET("/api/public/outbreak-resources", handler.ListResources)
	return router, db
}

func TestPublicOutbreakQuickResourceDiscovery(t *testing.T) {
	router, db := publicOutbreakTestRouter(t)
	now := time.Now().UTC().Add(-time.Minute)
	parent := models.Outbreak{Title: "Ebola response", SourceOrganization: "Ministry of Health", Status: "active", PublishedAt: &now, LastUpdate: now}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	resource := models.OutbreakResource{OutbreakID: parent.ID, Title: "Official response statement", Description: "Verified response announcement", ResourceType: "official_statement", URL: "https://health.go.ug/response", Status: "published", PublishedAt: &now}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/outbreak-resources?search=verified&target_type=external_url", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"target_type":"external_url"`) || !strings.Contains(response.Body.String(), `"reader_capability":"external_browser"`) || !strings.Contains(response.Body.String(), parent.Title) {
		t.Fatalf("quick resource response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestPublicOutbreakHandlerSupportsETagAndNotModified(t *testing.T) {
	router, db := publicOutbreakTestRouter(t)
	now := time.Now().UTC().Add(-time.Minute)
	item := models.Outbreak{Title: "Ebola response", Status: "active", GeographicArea: "Uganda", PublishedAt: &now, LastUpdate: now, VisualTone: "critical"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/public/outbreaks", nil))
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" || first.Header().Get("Last-Modified") == "" || first.Header().Get("Cache-Control") == "" {
		t.Fatalf("conditional headers missing: status=%d headers=%v body=%s", first.Code, first.Header(), first.Body.String())
	}
	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/public/outbreaks", nil)
	request.Header.Set("If-None-Match", first.Header().Get("ETag"))
	router.ServeHTTP(second, request)
	if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
		t.Fatalf("etag request status=%d body=%s", second.Code, second.Body.String())
	}
}

func TestPublicOutbreakHandlerRejectsInvalidTypedFilters(t *testing.T) {
	router, _ := publicOutbreakTestRouter(t)
	for _, path := range []string{
		"/api/public/outbreaks?region_id=not-a-uuid",
		"/api/public/outbreaks?effective_from=2026-08-02&effective_to=2026-08-01",
		"/api/public/outbreaks?updated_from=not-a-date",
		"/api/public/outbreaks?sort=deleted_at",
		"/api/public/outbreaks?order=random",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}
