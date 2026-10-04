package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestMigrateAndPersistApplicationData(t *testing.T) {
	database, err := OpenInMemory("store-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := database.PutSetting(context.Background(), "example", json.RawMessage(`{"enabled":true}`)); err != nil {
		t.Fatal(err)
	}
	value, err := database.GetSetting(context.Background(), "example")
	if err != nil || string(value) != `{"enabled":true}` {
		t.Fatalf("unexpected setting: %s, %v", value, err)
	}

	now := time.Now().UTC()
	root, err := database.CreateSearchRoot(context.Background(), SearchRoot{
		Name: "Workspace", Path: t.TempDir(), Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := database.GetSearchRoot(context.Background(), root.ID)
	if err != nil || loaded.Name != root.Name {
		t.Fatalf("unexpected root: %+v, %v", loaded, err)
	}

	raw := RawStatRecord{
		Source: "opencode", Project: "project", PeriodStart: now,
		PeriodEnd: now.Add(time.Hour), Granularity: "raw", FormatVersion: 1,
		ContentHash: "hash", Payload: json.RawMessage(`{"sessions":1}`),
	}
	_, created, err := database.UpsertRawStat(context.Background(), raw)
	if err != nil || !created {
		t.Fatalf("first raw insert: created=%t err=%v", created, err)
	}
	_, created, err = database.UpsertRawStat(context.Background(), raw)
	if err != nil || created {
		t.Fatalf("second raw insert was not idempotent: created=%t err=%v", created, err)
	}
	updated := raw
	updated.PeriodEnd = now.Add(2 * time.Hour)
	updated.ContentHash = "new-hash"
	updated.Payload = json.RawMessage(`{"sessions":2}`)
	_, created, err = database.UpsertRawStat(context.Background(), updated)
	if err != nil || created {
		t.Fatalf("raw refresh was not idempotent: created=%t err=%v", created, err)
	}
	rawRows, err := database.ListRawStats(context.Background(), "opencode", "project", nil)
	if err != nil || len(rawRows) != 1 || string(rawRows[0].Payload) != `{"sessions":2}` {
		t.Fatalf("expected one refreshed raw row, got %+v, %v", rawRows, err)
	}

	aggregate := Aggregate{
		Source: "opencode", Project: "project", PeriodStart: now,
		PeriodEnd: now.Add(time.Hour), Granularity: "daily", FormatVersion: 1,
		Sessions: 1, Models: json.RawMessage(`[]`), Tools: json.RawMessage(`{}`), Activity: json.RawMessage(`[]`),
	}
	_, created, err = database.UpsertAggregate(context.Background(), aggregate)
	if err != nil || !created {
		t.Fatalf("first aggregate insert: created=%t err=%v", created, err)
	}
	aggregate.PeriodEnd = now.Add(2 * time.Hour)
	aggregate.Sessions = 2
	_, created, err = database.UpsertAggregate(context.Background(), aggregate)
	if err != nil || created {
		t.Fatalf("aggregate refresh was not idempotent: created=%t err=%v", created, err)
	}
	aggregates, err := database.ListAggregates(context.Background(), "opencode", "project", "daily", nil, nil)
	if err != nil || len(aggregates) != 1 || aggregates[0].Sessions != 2 {
		t.Fatalf("expected one refreshed aggregate, got %+v, %v", aggregates, err)
	}
}

func TestShortcutsCanBeCreatedUpdatedAndTracked(t *testing.T) {
	database, err := OpenInMemory("shortcut-store-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	shortcut, err := database.CreateShortcut(context.Background(), Shortcut{
		Title: "Documentation", URL: "https://example.com/docs", Description: "Référence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if shortcut.ID == "" || shortcut.UsageCount != 0 {
		t.Fatalf("unexpected created shortcut: %+v", shortcut)
	}

	updated, err := database.UpdateShortcut(context.Background(), Shortcut{
		ID: shortcut.ID, Title: "Documentation mise à jour", URL: shortcut.URL,
		Description: "Nouvelle référence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Documentation mise à jour" || updated.Description != "Nouvelle référence" {
		t.Fatalf("unexpected updated shortcut: %+v", updated)
	}

	used, err := database.RecordShortcutUse(context.Background(), shortcut.ID)
	if err != nil {
		t.Fatal(err)
	}
	if used.UsageCount != 1 || used.LastUsedAt == nil {
		t.Fatalf("shortcut use was not recorded: %+v", used)
	}

	popular, err := database.ListShortcuts(context.Background(), "popular", 10)
	if err != nil || len(popular) != 1 || popular[0].ID != shortcut.ID {
		t.Fatalf("unexpected popular shortcuts: %+v, %v", popular, err)
	}
}

func TestShortcutsCanBeReordered(t *testing.T) {
	database, err := OpenInMemory("shortcut-order-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	first, err := database.CreateShortcut(context.Background(), Shortcut{Title: "First", URL: "https://example.com/first"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.CreateShortcut(context.Background(), Shortcut{Title: "Second", URL: "https://example.com/second"})
	if err != nil {
		t.Fatal(err)
	}
	third, err := database.CreateShortcut(context.Background(), Shortcut{Title: "Third", URL: "https://example.com/third"})
	if err != nil {
		t.Fatal(err)
	}
	folder, err := database.CreateShortcutFolder(context.Background(), ShortcutFolder{Name: "Documentation"})
	if err != nil {
		t.Fatal(err)
	}

	if err := database.ReorderShortcuts(context.Background(), []ShortcutOrder{
		{ID: third.ID, FolderID: &folder.ID}, {ID: first.ID}, {ID: second.ID},
	}); err != nil {
		t.Fatal(err)
	}
	ordered, err := database.ListShortcuts(context.Background(), "custom", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].ID != third.ID || ordered[1].ID != first.ID || ordered[2].ID != second.ID {
		t.Fatalf("unexpected custom shortcut order: %+v", ordered)
	}
	if ordered[0].FolderID == nil || *ordered[0].FolderID != folder.ID {
		t.Fatalf("shortcut folder was not persisted: %+v", ordered[0])
	}
	if err := database.DeleteShortcutFolder(context.Background(), folder.ID); err != nil {
		t.Fatal(err)
	}
	unassigned, err := database.GetShortcut(context.Background(), third.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unassigned.FolderID != nil {
		t.Fatalf("shortcut remained assigned after folder deletion: %+v", unassigned)
	}
}
