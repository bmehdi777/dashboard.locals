package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dashboard.locals/internal/config"
	"dashboard.locals/internal/search"
	"dashboard.locals/internal/shortcuts"
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

func TestShortcutsAPIProvidesCRUDAndUsageTracking(t *testing.T) {
	database, err := store.OpenInMemory("api-shortcuts-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(Dependencies{Shortcuts: shortcuts.NewService(database)})

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/shortcuts", strings.NewReader(`{"title":"Documentation","url":"https://example.com/docs","description":"Référence"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", create.Code, create.Body.String())
	}
	var created store.Shortcut
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Title != "Documentation" {
		t.Fatalf("unexpected shortcut: %+v", created)
	}

	use := httptest.NewRecorder()
	handler.ServeHTTP(use, httptest.NewRequest(http.MethodPost, "/api/v1/shortcuts/"+created.ID+"/use", nil))
	if use.Code != http.StatusOK {
		t.Fatalf("use status = %d body=%s", use.Code, use.Body.String())
	}
	var used store.Shortcut
	if err := json.Unmarshal(use.Body.Bytes(), &used); err != nil {
		t.Fatal(err)
	}
	if used.UsageCount != 1 || used.LastUsedAt == nil {
		t.Fatalf("usage was not tracked: %+v", used)
	}

	update := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPatch, "/api/v1/shortcuts/"+created.ID, strings.NewReader(`{"title":"Docs"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(update, request)
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), `"title":"Docs"`) {
		t.Fatalf("update response = %d %s", update.Code, update.Body.String())
	}

	remove := httptest.NewRecorder()
	handler.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/v1/shortcuts/"+created.ID, nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", remove.Code, remove.Body.String())
	}
}

func TestSearchStreamAPIEmitsResultAndDoneEvents(t *testing.T) {
	database, err := store.OpenInMemory("api-stream-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, "stream.go"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := database.CreateSearchRoot(context.Background(), store.SearchRoot{Name: "root", Path: rootPath, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	settings := config.NewService(database)
	searchService := search.NewService(database, apiStreamRunner{}, 10, time.Second)
	handler := NewHandler(Dependencies{
		Settings: settings,
		Roots:    search.NewRootService(database),
		Search:   searchService,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search/stream", strings.NewReader(`{"rootId":"`+root.ID+`","query":"needle"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/x-ndjson") {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	lines := strings.Split(strings.TrimSpace(recorder.Body.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected result and done events, got %q", recorder.Body.String())
	}
	var resultEvent, doneEvent map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &resultEvent); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &doneEvent); err != nil {
		t.Fatal(err)
	}
	if resultEvent["type"] != "result" || doneEvent["type"] != "done" {
		t.Fatalf("unexpected events: %v / %v", resultEvent, doneEvent)
	}
}

type apiStreamRunner struct{}

func (apiStreamRunner) Run(context.Context, string, string, []string) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (apiStreamRunner) RunStream(_ context.Context, _ string, _ string, _ []string, onLine func([]byte) error) error {
	event := `{"type":"match","data":{"path":{"text":"stream.go"},"lines":{"text":"needle\n"},"line_number":1,"submatches":[{"start":0}]}}`
	return onLine([]byte(event + "\n"))
}
