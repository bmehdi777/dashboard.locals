package launcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dashboard.locals/internal/store"
)

func TestOpenReplacesOnlyAllowedPlaceholders(t *testing.T) {
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	database, err := store.OpenInMemory("launcher-test")
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
	runner := &captureRunner{}
	service := NewService(database, runner)
	err = service.Open(context.Background(), root.ID, "main.go", 12, 4, Config{
		Command:   "code",
		Arguments: []string{"--goto", "{file}:{line}:{column}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if runner.command != "code" || len(runner.args) != 2 {
		t.Fatalf("unexpected command: %q %v", runner.command, runner.args)
	}
	if runner.args[1] != filepath.Join(rootPath, "main.go")+":12:4" {
		t.Fatalf("unexpected placeholder replacement: %q", runner.args[1])
	}
}

func TestValidateConfigRejectsUnknownPlaceholder(t *testing.T) {
	if err := ValidateConfig(Config{Command: "code", Arguments: []string{"{unknown}"}}); err == nil {
		t.Fatal("expected unknown placeholder to be rejected")
	}
}

type captureRunner struct {
	command string
	args    []string
}

func (r *captureRunner) Start(_ context.Context, command string, args []string) error {
	r.command = command
	r.args = append([]string(nil), args...)
	return nil
}
