package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseServiceFileBuildsAuthenticatedConnection(t *testing.T) {
	file, err := ParseServiceFile([]byte(`{"url":"http://127.0.0.1:4096","username":"user","password":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := connectionFromServiceFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if connection.BaseURL != "http://127.0.0.1:4096" || connection.authorization == "" {
		t.Fatalf("unexpected connection: %+v", connection)
	}
}

func TestDetectorAndStatsClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/info":
			_, _ = w.Write([]byte(`{"version":"1.2.3","pid":1,"urls":[],"paths":{"tmp":"/tmp"}}`))
		case "/api/experimental/session/stats":
			_, _ = w.Write([]byte(`{"data":{"range":{"from":1700000000000,"to":1700003600000},"sessions":2,"subagents":1,"prompts":3,"steps":4,"tokens":{"input":10,"output":20,"reasoning":5,"cache":{"read":6,"write":7}},"cost":0.12,"tools":{"mode":"summary","totals":{"calls":1,"succeeded":1,"failed":0,"unfinished":0}},"activeDays":1,"streak":1,"activity":[],"models":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	servicePath := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(servicePath, []byte(`{"url":"`+server.URL+`","token":"test-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	detector := NewDetector(servicePath, time.Second)
	detector.LookPath = func(string) (string, error) { return "/usr/bin/opencode", nil }
	detection := detector.Detect(context.Background())
	if detection.State != StateAvailable || detection.Version != "1.2.3" {
		t.Fatalf("unexpected detection: %+v", detection)
	}
	client, _, err := detector.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	from := time.UnixMilli(1700000000000).UTC()
	to := time.UnixMilli(1700003600000).UTC()
	snapshot, err := client.FetchStats(context.Background(), StatsQuery{From: &from, To: &to, Tools: "summary"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Sessions != 2 || snapshot.Tokens.CacheRead != 6 || snapshot.Tokens.CacheWrite != 7 || snapshot.Cost != 0.12 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if !json.Valid(snapshot.Payload) {
		t.Fatal("snapshot payload is not valid JSON")
	}
}

func TestDetectorReportsAbsentService(t *testing.T) {
	detector := NewDetector(filepath.Join(t.TempDir(), "missing.json"), time.Second)
	detection := detector.Detect(context.Background())
	if detection.State != StateAbsent {
		t.Fatalf("unexpected state: %+v", detection)
	}
}
