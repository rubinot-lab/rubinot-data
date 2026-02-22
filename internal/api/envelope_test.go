package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestSuccessEnvelope(t *testing.T) {
	t.Setenv("APP_VERSION", "v0.2.0")
	t.Setenv("APP_COMMIT", "abc1234")

	payload := gin.H{"name": "Belaria"}
	envelope := successEnvelope("world", payload, []string{"https://www.rubinot.com.br/?subtopic=worlds&world=Belaria"})

	info, ok := envelope["information"].(information)
	if !ok {
		t.Fatalf("information payload has unexpected type %T", envelope["information"])
	}
	if info.API.Version != 1 {
		t.Fatalf("expected api version 1, got %d", info.API.Version)
	}
	if info.API.Release != "v0.2.0" {
		t.Fatalf("unexpected release value: %s", info.API.Release)
	}
	if info.API.Commit != "abc1234" {
		t.Fatalf("unexpected commit value: %s", info.API.Commit)
	}
	if info.Status.HTTPCode != 200 {
		t.Fatalf("expected status code 200, got %d", info.Status.HTTPCode)
	}
	if info.Status.Message != "ok" {
		t.Fatalf("expected status message ok, got %q", info.Status.Message)
	}
	if _, err := time.Parse(time.RFC3339, info.Timestamp); err != nil {
		t.Fatalf("timestamp must be RFC3339, got %q: %v", info.Timestamp, err)
	}
	if len(info.Sources) != 1 {
		t.Fatalf("expected one source URL, got %d", len(info.Sources))
	}

	gotPayload, ok := envelope["world"].(gin.H)
	if !ok {
		t.Fatalf("world payload has unexpected type %T", envelope["world"])
	}
	if !reflect.DeepEqual(gotPayload, payload) {
		t.Fatalf("payload mismatch: got %+v want %+v", gotPayload, payload)
	}
}

func TestErrorEnvelope(t *testing.T) {
	t.Setenv("APP_VERSION", "v0.2.0")
	t.Setenv("APP_COMMIT", "abc1234")

	envelope := errorEnvelope(404, 20004, "character not found", nil)
	info, ok := envelope["information"].(information)
	if !ok {
		t.Fatalf("information payload has unexpected type %T", envelope["information"])
	}

	if info.Status.HTTPCode != 404 {
		t.Fatalf("expected status code 404, got %d", info.Status.HTTPCode)
	}
	if info.Status.Error != 20004 {
		t.Fatalf("expected error code 20004, got %d", info.Status.Error)
	}
	if info.Status.Message != "character not found" {
		t.Fatalf("unexpected message: %q", info.Status.Message)
	}
	if len(info.Sources) != 0 {
		t.Fatalf("expected empty sources list, got %d", len(info.Sources))
	}
}

func TestHandleEndpointMapsValidationError(t *testing.T) {
	t.Setenv("APP_VERSION", "v0.2.0")
	t.Setenv("APP_COMMIT", "abc1234")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/test", handleEndpoint(func(_ *gin.Context) (endpointResult, error) {
		return endpointResult{Sources: []string{"https://www.rubinot.com.br"}}, validation.NewError(validation.ErrorFlareSolverrTimeout, "flaresolverr timeout", nil)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	info, ok := body["information"].(map[string]any)
	if !ok {
		t.Fatalf("missing information object in response: %+v", body)
	}
	status, ok := info["status"].(map[string]any)
	if !ok {
		t.Fatalf("missing status object in response: %+v", info)
	}
	if int(status["http_code"].(float64)) != http.StatusGatewayTimeout {
		t.Fatalf("unexpected http_code payload: %+v", status)
	}
	if int(status["error"].(float64)) != validation.ErrorFlareSolverrTimeout {
		t.Fatalf("unexpected error payload: %+v", status)
	}
}
