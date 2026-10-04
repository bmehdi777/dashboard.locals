package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dashboard.locals/internal/store"
)

func TestBuildArgumentsKeepsUserQueryAsAnArgument(t *testing.T) {
	args, err := BuildArguments(Options{Query: "--danger", Literal: true, RespectGitignore: false, IncludeBinary: true})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--fixed-strings") || !strings.Contains(joined, "--no-ignore") || !strings.Contains(joined, "--text") {
		t.Fatalf("missing expected options: %v", args)
	}
	if args[len(args)-2] != "--danger" || args[len(args)-1] != "." {
		t.Fatalf("query was not kept as a positional argument: %v", args)
	}
}

func TestResolveResultPathRejectsTraversalAndSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveResultPath(root, "../outside.go"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	target := filepath.Join(root, "file.go")
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolveResultPath(root, "link.go"); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}

func TestSearchParsesMatchesAndHonorsLimit(t *testing.T) {
	rootPath := t.TempDir()
	filePath := filepath.Join(rootPath, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	database, err := store.OpenInMemory("search-test")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	root, err := database.CreateSearchRoot(context.Background(), store.SearchRoot{Name: "root", Path: rootPath, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	runner := runnerFunc(func(_ context.Context, _ string, _ string, _ []string) ([]byte, []byte, error) {
		event := map[string]any{
			"type": "match",
			"data": map[string]any{
				"path":        map[string]string{"text": "main.go"},
				"lines":       map[string]string{"text": "package main\n"},
				"line_number": 1,
				"submatches":  []map[string]int{{"start": 8}},
			},
		}
		data, _ := json.Marshal(event)
		return append(data, '\n'), nil, nil
	})
	service := NewService(database, runner, 1, time.Second)
	results, err := service.Search(context.Background(), root.ID, Options{Query: "main", MaxResults: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != "main.go" || results[0].Line != 1 || results[0].Column != 9 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

type runnerFunc func(context.Context, string, string, []string) ([]byte, []byte, error)

func (f runnerFunc) Run(ctx context.Context, dir, executable string, args []string) ([]byte, []byte, error) {
	return f(ctx, dir, executable, args)
}
