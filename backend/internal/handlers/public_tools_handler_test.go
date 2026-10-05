package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mediguide/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestGuestCalculatorReads(t *testing.T) {
	h, db := testCalculatorHandler(t)
	r := gin.New()
	r.GET("/calculators", h.PublicList)
	r.GET("/calculators/:id", h.PublicGet)
	r.GET("/calculators/:id/definition", h.PublicDefinition)
	r.GET("/calculators/:id/content", h.PublicContent)
	for _, kind := range []string{"calculator", "decision_tool", "checklist"} {
		for _, status := range []string{"active", "draft", "archived"} {
			tool := models.Calculator{AddedByUserID: uuid.New(), Name: kind + status, Type: kind, Status: status, Version: "1", AppFileJSON: datatypes.JSON(`{}`)}
			if err := db.Create(&tool).Error; err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/calculators/"+tool.ID.String(), nil))
			want := http.StatusNotFound
			if status == "active" {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Fatalf("%s %s: got %d, want %d: %s", kind, status, response.Code, want, response.Body.String())
			}
			if status != "active" {
				for _, suffix := range []string{"/definition", "/content"} {
					response = httptest.NewRecorder()
					r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/calculators/"+tool.ID.String()+suffix, nil))
					if response.Code != http.StatusNotFound {
						t.Fatalf("non-public tool %s returned %d", suffix, response.Code)
					}
				}
			}
		}
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/calculators?type="+kind+"&status=draft,archived", nil))
		var envelope struct {
			Data struct {
				Items []models.Calculator `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || len(envelope.Data.Items) != 1 || envelope.Data.Items[0].Status != "active" || envelope.Data.Items[0].Type != kind {
			t.Fatalf("guest catalogue filter failed: %s", response.Body.String())
		}
	}
}

func TestGuestDirectoryReadsWithoutClaims(t *testing.T) {
	h := testContentReferenceHandler(t)
	r := gin.New()
	r.GET("/ministry-directory", h.PublicListDirectory)
	r.GET("/ministry-directory/:id", h.PublicGetDirectory)
	for _, status := range []string{"active", "inactive", "pending"} {
		entry := models.MinistryDirectoryEntry{Name: "Contact " + status, Status: status}
		if err := h.Service.DB.Create(&entry).Error; err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ministry-directory/"+entry.ID.String(), nil))
		want := http.StatusNotFound
		if status == "active" {
			want = http.StatusOK
		}
		if response.Code != want {
			t.Fatalf("directory %s: got %d, want %d: %s", status, response.Code, want, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ministry-directory?status=pending", nil))
	var envelope struct {
		Data struct {
			Items []models.MinistryDirectoryEntry `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(envelope.Data.Items) != 1 || envelope.Data.Items[0].Status != "active" {
		t.Fatalf("guest directory filter failed: %s", response.Body.String())
	}
}
