package shortcuts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"dashboard.locals/internal/store"
)

func TestServiceValidatesHTTPShortcutURLs(t *testing.T) {
	database, err := store.OpenInMemory("shortcut-service-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)

	if _, err := service.Create(context.Background(), "Script", "javascript:alert(1)", "", nil); !errors.Is(err, ErrInvalidShortcut) {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
	if _, err := service.Create(context.Background(), "Script", "https://example.com", "", nil); err != nil {
		t.Fatal(err)
	}
}

func TestServiceUpdatesAndUsesShortcut(t *testing.T) {
	database, err := store.OpenInMemory("shortcut-service-use-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)

	created, err := service.Create(context.Background(), "Accueil", "https://example.com", "Portail", nil)
	if err != nil {
		t.Fatal(err)
	}
	description := "Portail principal"
	updated, err := service.Update(context.Background(), created.ID, nil, nil, &description, nil)
	if err != nil || updated.Description != description {
		t.Fatalf("unexpected update: %+v, %v", updated, err)
	}
	used, err := service.Use(context.Background(), created.ID)
	if err != nil || used.UsageCount != 1 {
		t.Fatalf("unexpected usage: %+v, %v", used, err)
	}
}

func TestServiceFetchesAndCachesShortcutFavicon(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nicon"))
	}))
	defer server.Close()

	database, err := store.OpenInMemory("shortcut-favicon-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWithHTTPClient(database, server.Client())
	shortcut, err := service.Create(context.Background(), "Site", server.URL+"/page", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	data, contentType, err := service.Favicon(context.Background(), shortcut.ID)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "image/png" || len(data) == 0 {
		t.Fatalf("unexpected favicon response: contentType=%q bytes=%d", contentType, len(data))
	}
	if _, _, err := service.Favicon(context.Background(), shortcut.ID); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("favicon was not cached, requests=%d", requests.Load())
	}
}

func TestFaviconRedirectAllowsWWWAlias(t *testing.T) {
	left, err := url.Parse("https://google.fr/favicon.ico")
	if err != nil {
		t.Fatal(err)
	}
	right, err := url.Parse("https://www.google.fr/favicon.ico")
	if err != nil {
		t.Fatal(err)
	}
	if !sameHTTPOrigin(left, right) {
		t.Fatal("expected www alias to be accepted for favicon redirects")
	}
}
