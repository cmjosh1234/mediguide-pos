package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"mediguide/internal/services"

	"github.com/gin-gonic/gin"
)

func TestOutbreakAdminValidationErrorCarriesFieldsInMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	respond := func(err error) (int, map[string]any) {
		t.Helper()
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		OutbreakAdminHandler{}.result(c, http.StatusOK, nil, err)
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return recorder.Code, body
	}

	code, body := respond(&services.OutbreakValidationError{
		Message: "Effective at can't be earlier than the start date.",
		Fields: []services.OutbreakFieldError{
			{Field: "effective_at", Message: "Effective at can't be earlier than the start date."},
			{Field: "start_date", Message: "Effective at can't be earlier than the start date."},
		},
	})
	meta, _ := body["meta"].(map[string]any)
	fields, _ := meta["fields"].([]any)
	if code != http.StatusBadRequest || body["error"] != "Effective at can't be earlier than the start date." || len(fields) != 2 {
		t.Fatalf("field error response: %d %#v", code, body)
	}
	if first, _ := fields[0].(map[string]any); first["field"] != "effective_at" || first["message"] == "" {
		t.Fatalf("first field: %#v", fields[0])
	}

	code, body = respond(&services.OutbreakValidationError{Message: "Only a draft can be submitted for review."})
	if code != http.StatusBadRequest || body["error"] != "Only a draft can be submitted for review." || body["meta"] != nil {
		t.Fatalf("a reason without fields should not carry meta: %d %#v", code, body)
	}

	if code, _ = respond(errors.Join(services.ErrOutbreakConflict)); code != http.StatusConflict {
		t.Fatalf("conflict status: %d", code)
	}
}
