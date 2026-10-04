package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientRejectsUnsafeServerURLs(t *testing.T) {
	tests := []string{
		"",
		"127.0.0.1:8080",
		"ftp://127.0.0.1:8080",
		"http://user:password@127.0.0.1:8080",
		"http://127.0.0.1:8080/?token=secret",
	}
	for _, rawURL := range tests {
		if _, err := NewClient(rawURL, time.Second); !errors.Is(err, ErrUsage) {
			t.Errorf("NewClient(%q) error = %v, want ErrUsage", rawURL, err)
		}
	}
}

func TestClientDecodesStructuredAPIErrorWithoutRawBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusConflict)
		_, _ = writer.Write([]byte(`{"error":{"code":"conflict","message":"resource already exists"},"secret":"do-not-print"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var response any
	err = client.Do(context.Background(), http.MethodGet, "/api/v1/health", nil, nil, &response)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Status != http.StatusConflict || apiErr.Code != "conflict" || apiErr.Message != "resource already exists" {
		t.Fatalf("unexpected API error: %+v", apiErr)
	}
	if strings.Contains(err.Error(), "do-not-print") {
		t.Fatalf("raw response leaked in error: %v", err)
	}
}

func TestClientHonorsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = client.Do(ctx, http.MethodGet, "/api/v1/health", nil, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestClientRejectsOversizedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`"` + strings.Repeat("x", maxResponseSize) + `"`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var response json.RawMessage
	err = client.Do(context.Background(), http.MethodGet, "/api/v1/health", nil, nil, &response)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}
