package shortcuts

import (
	"context"
	"errors"
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

	if _, err := service.Create(context.Background(), "Script", "javascript:alert(1)", ""); !errors.Is(err, ErrInvalidShortcut) {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
	if _, err := service.Create(context.Background(), "Script", "https://example.com", ""); err != nil {
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

	created, err := service.Create(context.Background(), "Accueil", "https://example.com", "Portail")
	if err != nil {
		t.Fatal(err)
	}
	description := "Portail principal"
	updated, err := service.Update(context.Background(), created.ID, nil, nil, &description)
	if err != nil || updated.Description != description {
		t.Fatalf("unexpected update: %+v, %v", updated, err)
	}
	used, err := service.Use(context.Background(), created.ID)
	if err != nil || used.UsageCount != 1 {
		t.Fatalf("unexpected usage: %+v, %v", used, err)
	}
}
