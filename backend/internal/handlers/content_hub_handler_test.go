package handlers

import (
	"fmt"
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

// The dashboard archives a pillar by sending its stored fields with only the
// status changed; a body without the name is refused outright.
func TestArchivingAPillarNeedsItsStoredFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.New())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.ContentHub{}, &models.ContentPillar{}); err != nil {
		t.Fatal(err)
	}
	hub := models.ContentHub{Name: "Malaria", Slug: "malaria", Audience: "all", Status: models.ContentHubStatusActive, LockVersion: 1}
	if err := db.Create(&hub).Error; err != nil {
		t.Fatal(err)
	}
	pillar := models.ContentPillar{HubID: hub.ID, Name: "Overview", Slug: "overview", Description: "Screening and diagnosis overview.", Icon: "book-open", Color: "clinical", SortOrder: 10, Status: models.ContentPillarStatusActive, LockVersion: 1}
	if err := db.Create(&pillar).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.Nil})
		c.Next()
	})
	router.PATCH("/content-hubs/:id/pillars/:pillarId", ContentHubHandler{Service: services.ContentHubService{DB: db}}.UpdatePillar)
	patch := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/content-hubs/%s/pillars/%s", hub.ID, pillar.ID), strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	if response := patch(`{"status":"archived","lock_version":1}`); response.Code != http.StatusBadRequest {
		t.Fatalf("expected the old status-only body to be refused, got %d %s", response.Code, response.Body.String())
	}
	response := patch(`{"name":"Overview","slug":"overview","description":"Screening and diagnosis overview.","icon":"book-open","color":"clinical","sort_order":10,"status":"archived","lock_version":1}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected the archive to succeed, got %d %s", response.Code, response.Body.String())
	}
	var stored models.ContentPillar
	if err := db.First(&stored, "id = ?", pillar.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.ContentPillarStatusArchived || stored.Description != pillar.Description || stored.Color != "clinical" || stored.SortOrder != 10 || stored.Slug != "overview" {
		t.Fatalf("archiving changed more than the status: %+v", stored)
	}
}
