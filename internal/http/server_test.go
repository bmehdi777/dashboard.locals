package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dashboard.locals/internal/http/api/v1"
)

func TestServerServesEmbeddedFrontendAndStructuredAPIErrors(t *testing.T) {
	handler := NewHandler(Config{API: v1.Dependencies{Version: "test"}})
	frontend := httptest.NewRecorder()
	handler.ServeHTTP(frontend, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if frontend.Code != http.StatusOK || !strings.Contains(frontend.Body.String(), "dashboard.locals") {
		t.Fatalf("unexpected frontend response: %d %s", frontend.Code, frontend.Body.String())
	}

	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil))
	if api.Code != http.StatusNotFound || !strings.Contains(api.Body.String(), `"error"`) {
		t.Fatalf("unexpected API response: %d %s", api.Code, api.Body.String())
	}

	notAPI := httptest.NewRecorder()
	handler.ServeHTTP(notAPI, httptest.NewRequest(http.MethodGet, "/api/v10/status", nil))
	if notAPI.Code != http.StatusOK {
		t.Fatalf("unexpected non-API response: %d %s", notAPI.Code, notAPI.Body.String())
	}
}
