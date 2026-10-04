package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runCLI(t *testing.T, server *httptest.Server, args ...string) (string, string, error) {
	t.Helper()
	var output, errorsOutput bytes.Buffer
	client, err := NewClientWithHTTP(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand(RootOptions{Out: &output, ErrOut: &errorsOutput, Client: client})
	command.SetArgs(args)
	_, err = command.ExecuteC()
	return output.String(), errorsOutput.String(), err
}

func TestSearchRunSendsOnlyExplicitOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/search" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["rootId"] != "root-1" || body["query"] != "TODO" {
			t.Fatalf("unexpected required fields: %#v", body)
		}
		for key, expected := range map[string]any{
			"literal":          true,
			"respectGitignore": false,
			"includeBinary":    true,
			"maxResults":       float64(3),
			"timeoutMs":        float64(2000),
		} {
			if body[key] != expected {
				t.Errorf("body[%q] = %#v, want %#v", key, body[key], expected)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"rootId":"root-1","count":1,"results":[{"path":"main.go","line":4,"column":2,"excerpt":"TODO"}]}`))
	}))
	defer server.Close()

	output, errorOutput, err := runCLI(t, server, "search", "run", "TODO", "--root", "root-1", "--literal", "--no-gitignore", "--include-binary", "--max-results", "3", "--search-timeout", "2s")
	if err != nil {
		t.Fatalf("command error = %v", err)
	}
	if errorOutput != "" {
		t.Fatalf("unexpected stderr: %s", errorOutput)
	}
	if !strings.Contains(output, "main.go:4:2: TODO") || !strings.Contains(output, "1 result(s)") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestSearchRootsListSupportsJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/search-roots" || request.URL.Query().Get("includeDisabled") != "true" {
			t.Fatalf("unexpected request: %s?%s", request.URL.Path, request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"root-1","name":"workspace","path":"/tmp/workspace","enabled":false}]}`))
	}))
	defer server.Close()

	output, _, err := runCLI(t, server, "--json", "search", "roots", "list", "--include-disabled")
	if err != nil {
		t.Fatalf("command error = %v", err)
	}
	var response rootsResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output)
	}
	if len(response.Data) != 1 || response.Data[0].ID != "root-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestConfigPatchReadsStdinAndUsesPatchMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch || request.URL.Path != "/api/v1/settings" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["search"].(map[string]any)["literal"] != true {
			t.Fatalf("unexpected patch: %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"search":{"literal":true}}`))
	}))
	defer server.Close()

	var output, errorOutput bytes.Buffer
	client, err := NewClientWithHTTP(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand(RootOptions{
		In: strings.NewReader(`{"search":{"literal":true}}`), Out: &output, ErrOut: &errorOutput, Client: client,
	})
	command.SetArgs([]string{"config", "patch", "--file", "-"})
	if _, err := command.ExecuteC(); err != nil {
		t.Fatalf("command error = %v", err)
	}
	if errorOutput.Len() != 0 || !strings.Contains(output.String(), `"literal": true`) {
		t.Fatalf("unexpected output: stdout=%s stderr=%s", output.String(), errorOutput.String())
	}
}

func TestStatsCompactSendsExplicitDropRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/stats/compact" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var body compactStatsRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Before != "2026-01-01T00:00:00Z" || !body.DryRun || !body.DropRaw || body.Granularity != "daily" {
			t.Fatalf("unexpected request body: %+v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"source":"opencode","granularity":"daily","dryRun":true,"rawRead":2,"aggregatesCreated":1,"aggregatesUpdated":0,"rawDeleted":0}`))
	}))
	defer server.Close()

	output, _, err := runCLI(t, server, "stats", "compact", "--before", "2026-01-01T00:00:00Z", "--granularity", "daily", "--dry-run", "--drop-raw")
	if err != nil {
		t.Fatalf("command error = %v", err)
	}
	if !strings.Contains(output, "dry-run: true") || !strings.Contains(output, "raw read: 2") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestCommandValidationAndExitCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":{"code":"invalid_request","message":"bad input"}}`))
	}))
	defer server.Close()

	_, _, err := runCLI(t, server, "search", "run", "query")
	if err == nil || ExitCode(err) != exitUsage {
		t.Fatalf("missing root error = %v, exit code = %d", err, ExitCode(err))
	}
	_, _, err = runCLI(t, server, "server", "health")
	if err == nil || ExitCode(err) != exitAPI {
		t.Fatalf("API error = %v, exit code = %d", err, ExitCode(err))
	}
	if got := fmt.Sprint(err); !strings.Contains(got, "invalid_request") {
		t.Fatalf("API error does not contain code: %s", got)
	}
}
