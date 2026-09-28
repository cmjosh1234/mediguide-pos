package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mediguide/internal/requestctx"

	"github.com/gin-gonic/gin"
)

func TestRequestIDPreservesValidIncomingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, requestctx.CorrelationID(c.Request.Context()))
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "request-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get(RequestIDHeader); got != "request-123" {
		t.Fatalf("unexpected response request id: %q", got)
	}
	if got := response.Body.String(); got != "request-123" {
		t.Fatalf("request id was not propagated into context: %q", got)
	}
}

func TestRequestIDGeneratesIDWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected generated request id")
	}
}

func TestRequestIDReplacesUnsafeIncomingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "unsafe id with spaces")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if got := response.Header().Get(RequestIDHeader); got == "" || got == "unsafe id with spaces" {
		t.Fatalf("expected unsafe request id to be replaced, got %q", got)
	}
}
