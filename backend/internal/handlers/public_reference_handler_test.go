package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"mediguide/internal/models"
	"mediguide/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGuestDrugReadsOnlyActiveApprovedRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Drug{}, &models.DrugClass{}, &models.TherapeuticCategory{}, &models.DrugCategory{}, &models.DrugTag{}); err != nil {
		t.Fatal(err)
	}
	h := DrugHandler{Service: services.DrugService{DB: db}}
	r := gin.New()
	r.GET("/drugs", h.PublicList)
	r.GET("/drugs/:id", h.PublicGet)
	for _, pair := range [][2]string{{"active", "approved"}, {"active", "pending"}, {"inactive", "approved"}, {"archived", "approved"}} {
		drug := models.Drug{Name: pair[0] + pair[1], Status: pair[0], ReviewStatus: pair[1]}
		if err := db.Create(&drug).Error; err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/drugs/"+drug.ID.String(), nil))
		want := 404
		if pair[0] == "active" && pair[1] == "approved" {
			want = 200
		}
		if response.Code != want {
			t.Fatalf("%v got %d want %d: %s", pair, response.Code, want, response.Body)
		}
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/drugs?status=inactive&review_status=pending", nil))
	var result struct {
		Data struct {
			Items []models.Drug `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(result.Data.Items) != 1 || result.Data.Items[0].ReviewStatus != "approved" {
		t.Fatalf("public filters: %s", response.Body)
	}
}

func TestGuestAbbreviationReadsWithoutClaims(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Abbreviation{}, &models.GuidelineCategory{}, &models.GuidelineTag{}); err != nil {
		t.Fatal(err)
	}
	item := models.Abbreviation{Abbreviation: "BP", Meaning: "Blood pressure"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	h := GuidelineContentHandler{Service: services.GuidelineContentService{DB: db}}
	r := gin.New()
	r.GET("/abbreviations", h.PublicListAbbreviations)
	r.GET("/abbreviations/:id", h.PublicGetAbbreviation)
	r.GET("/categories", h.PublicListCategories)
	r.GET("/categories/:id", h.PublicGetCategory)
	for _, status := range []string{"active", "inactive"} {
		category := models.GuidelineCategory{Name: status, Status: status}
		if err := db.Create(&category).Error; err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/categories/"+category.ID.String(), nil))
		want := 404
		if status == "active" {
			want = 200
		}
		if response.Code != want {
			t.Fatalf("guest category %s: %d %s", status, response.Code, response.Body)
		}
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/categories?status=inactive", nil))
	if response.Code != 200 {
		t.Fatalf("guest category list: %d %s", response.Code, response.Body)
	}
	for _, path := range []string{"/abbreviations?search=BP", "/abbreviations/" + item.ID.String()} {
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 200 {
			t.Fatalf("guest %s returned %d: %s", path, response.Code, response.Body)
		}
	}
}
