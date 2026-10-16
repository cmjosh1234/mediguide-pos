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
)

func TestGuidelineWithdrawalPermissionValidationAndConflict(t *testing.T) {
	db := newGuidelineLibraryHandlerDB(t)
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	doc := createPublishedGuidelineLibraryHandlerDocument(t, db)
	docID := doc.ID
	handler := GuidelineHandler{Service: services.GuidelineService{DB: db}}
	for _, tt := range []struct {
		name, permission, id, body string
		want                       int
	}{
		{"reader forbidden", "guideline.read", docID.String(), `{}`, http.StatusForbidden},
		{"writer forbidden", "guideline.write", docID.String(), `{}`, http.StatusForbidden},
		{"invalid id", "guideline.publish", "bad-id", `{}`, http.StatusBadRequest},
		{"missing reason", "guideline.publish", docID.String(), `{}`, http.StatusBadRequest},
		{"blank reason", "guideline.publish", docID.String(), `{"reason":" ","current_version_id":"` + doc.CurrentVersionID.String() + `"}`, http.StatusBadRequest},
		{"stale version", "guideline.publish", docID.String(), `{"reason":"Review","current_version_id":"` + uuid.NewString() + `"}`, http.StatusConflict},
		{"missing document", "guideline.publish", uuid.NewString(), `{"reason":"Review","current_version_id":"` + uuid.NewString() + `"}`, http.StatusNotFound},
		{"published withdrawal", "guideline.publish", docID.String(), `{"reason":"Review","current_version_id":"` + doc.CurrentVersionID.String() + `"}`, http.StatusNoContent},
		{"already withdrawn", "guideline.publish", docID.String(), `{"reason":"Review","current_version_id":"` + doc.CurrentVersionID.String() + `"}`, http.StatusConflict},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(middleware.ClaimsKey, &security.Claims{UserID: uuid.New(), Perms: []string{tt.permission}})
				c.Next()
			})
			router.POST("/guidelines/:id/withdraw", middleware.RequirePermission("guideline.publish"), handler.Withdraw)
			request := httptest.NewRequest(http.MethodPost, "/guidelines/"+tt.id+"/withdraw", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.want {
				t.Fatalf("got %d, want %d: %s", response.Code, tt.want, response.Body.String())
			}
		})
	}
}
