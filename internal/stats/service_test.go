package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dashboard.locals/internal/opencode"
	"dashboard.locals/internal/store"
)

func TestSyncIsIdempotentAndCompactDryRunDoesNotDeleteRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/info" {
			_, _ = w.Write([]byte(`{"version":"test"}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"range":{"from":1700000000000,"to":1791139951873},"sessions":1,"subagents":0,"prompts":2,"steps":3,"tokens":{"input":4,"output":5,"reasoning":6,"cache":{"read":7,"write":8}},"cost":0.5,"tools":{"mode":"summary"},"activeDays":1,"streak":1,"activity":[],"models":[]}}`))
	}))
	defer server.Close()
	servicePath := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(servicePath, []byte(`{"url":"`+server.URL+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	detector := opencode.NewDetector(servicePath, time.Second)
	detector.LookPath = func(string) (string, error) { return "/bin/opencode", nil }
	database, err := store.OpenInMemory("stats-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(database, detector, "UTC", "summary", "daily")
	from := time.UnixMilli(1700000000000).UTC()
	to := time.UnixMilli(1700003600000).UTC()
	first, err := service.Sync(context.Background(), Query{From: &from, To: &to})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Sync(context.Background(), Query{From: &from, To: &to})
	if err != nil {
		t.Fatal(err)
	}
	if first.RawCreated != 1 || second.RawCreated != 0 || second.RawExisting != 1 {
		t.Fatalf("unexpected idempotence result: first=%+v second=%+v", first, second)
	}
	rawRows, err := database.ListRawStats(context.Background(), opencode.Source, "", nil)
	if err != nil || len(rawRows) != 1 {
		t.Fatalf("synchronization created duplicate raw rows: %d, %v", len(rawRows), err)
	}
	before := to.Add(time.Hour)
	compact, err := service.Compact(context.Background(), CompactRequest{Before: before, Granularity: "daily", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if compact.RawRead != 1 || compact.RawDeleted != 0 {
		t.Fatalf("unexpected dry-run result: %+v", compact)
	}
	raw, err := database.ListRawStats(context.Background(), opencode.Source, "", &before)
	if err != nil || len(raw) != 1 {
		t.Fatalf("dry-run deleted raw data: %d, %v", len(raw), err)
	}
	view, err := service.Get(context.Background(), Query{From: &from, To: &to})
	if err != nil || len(view.Aggregates) != 1 {
		t.Fatalf("unexpected stats view: %+v, %v", view, err)
	}
	if !json.Valid(view.Aggregates[0].Tools) {
		t.Fatal("aggregate tools are invalid JSON")
	}
}
