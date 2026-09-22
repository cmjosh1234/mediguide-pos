package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mediguide/internal/config"
	"mediguide/internal/models"
	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthHandlerLoginUsesV2Contract(t *testing.T) {
	handler := testAuthHandler(t)
	if _, err := handler.Service.Register(services.RegisterInput{
		Name:     "Admin",
		Email:    "admin@mediguide.local",
		Password: "secret",
	}); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/api/v2/auth/login", handler.Login)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/auth/login",
		strings.NewReader(`{"email":"admin@mediguide.local","password":"secret"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Token        string      `json:"token"`
			RefreshToken string      `json:"refresh_token"`
			SessionID    string      `json:"session_id"`
			User         models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || envelope.Data.Token == "" || envelope.Data.RefreshToken == "" || envelope.Data.SessionID == "" {
		t.Fatalf("incomplete login response: %#v", envelope)
	}
	if envelope.Data.User.Email != "admin@mediguide.local" {
		t.Fatalf("unexpected login user: %#v", envelope.Data.User)
	}
}

func TestAuthHandlerRegisterAcceptsMissingLicenseAndNormalizesMobilePayload(t *testing.T) {
	handler := testAuthHandler(t)
	router := gin.New()
	router.POST("/api/v2/auth/register", handler.Register)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/auth/register",
		strings.NewReader(`{
			"name":"Mobile User",
			"email":"Mobile.User@Example.Test",
			"password":"Password8",
			"preferred_language":"english",
			"specialization":["Internal Medicine"]
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Success bool        `json:"success"`
		Data    models.User `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success {
		t.Fatalf("unexpected registration response: %#v", envelope)
	}
	if envelope.Data.Email != "mobile.user@example.test" {
		t.Fatalf("email was not normalized: %q", envelope.Data.Email)
	}
	if envelope.Data.LicenseNumber != nil {
		t.Fatalf("expected an omitted license number, got %q", *envelope.Data.LicenseNumber)
	}
	if envelope.Data.PreferredLanguage == nil || *envelope.Data.PreferredLanguage != "English" {
		t.Fatalf("preferred language was not normalized: %#v", envelope.Data.PreferredLanguage)
	}
	if len(envelope.Data.Specialization) != 1 || envelope.Data.Specialization[0] != "Internal Medicine" {
		t.Fatalf("unexpected specialization: %#v", envelope.Data.Specialization)
	}

	if _, err := handler.Service.Login(" MOBILE.USER@EXAMPLE.TEST ", "Password8", services.RequestMetadata{}); err != nil {
		t.Fatalf("expected normalized email login to succeed: %v", err)
	}
}

func TestAuthHandlerEmailVerificationUsesNeutralTypedContract(t *testing.T) {
	handler := testAuthHandler(t)
	user, err := handler.Service.Register(services.RegisterInput{
		Name: "Verification User", Email: "verification@example.test", Password: "Password8",
	})
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/api/v2/auth/email-verification/request", handler.RequestEmailVerification)
	router.POST("/api/v2/auth/email-verification/confirm", handler.ConfirmEmailVerification)

	request := httptest.NewRequest(http.MethodPost, "/api/v2/auth/email-verification/request", strings.NewReader(`{"email":"verification@example.test"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected request status %d: %s", response.Code, response.Body.String())
	}
	var requested struct {
		Success bool                         `json:"success"`
		Data    services.AccountActionResult `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &requested); err != nil {
		t.Fatal(err)
	}
	if !requested.Success || !requested.Data.Accepted || requested.Data.DeliveryAccepted || requested.Data.DevelopmentToken == "" {
		t.Fatalf("unexpected verification response: %#v", requested)
	}

	confirm := httptest.NewRequest(http.MethodPost, "/api/v2/auth/email-verification/confirm", strings.NewReader(`{"token":"`+requested.Data.DevelopmentToken+`"}`))
	confirm.Header.Set("Content-Type", "application/json")
	confirmed := httptest.NewRecorder()
	router.ServeHTTP(confirmed, confirm)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("unexpected confirmation status %d: %s", confirmed.Code, confirmed.Body.String())
	}
	if err := handler.Service.DB.First(user, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !user.Verified {
		t.Fatal("expected confirmation endpoint to verify the user")
	}
}

func testAuthHandler(t *testing.T) AuthHandler {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.AuthSession{},
		&models.AccountActionToken{},
		&models.AuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	return AuthHandler{Service: services.AuthService{
		DB: database,
		Cfg: config.Config{
			AppEnv:               "development",
			JWTSecret:            "test-secret-that-is-long-enough-for-hmac",
			JWTIssuer:            "mediguide-test",
			JWTTTLMinutes:        15,
			JWTRefreshTTLMinutes: 60,
		},
	}}
}
