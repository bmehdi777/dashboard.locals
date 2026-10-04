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
}
