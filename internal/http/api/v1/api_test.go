package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dashboard.locals/internal/config"
	"dashboard.locals/internal/search"
	"dashboard.locals/internal/store"
)

func TestUnknownAPIRouteReturnsStructuredError(t *testing.T) {
	handler := NewHandler(Dependencies{Version: "test"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "not_found" {
		t.Fatalf("unexpected error: %+v", response)
	}
}

func TestSearchRootsAPIValidatesPaths(t *testing.T) {
	database, err := store.OpenInMemory("api-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	settings := config.NewService(database)
	handler := NewHandler(Dependencies{Settings: settings, Roots: search.NewRootService(database)})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search-roots", strings.NewReader(`{"name":"bad","path":"/does/not/exist"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}
