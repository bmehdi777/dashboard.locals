package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCreatesAccessAndErrorLogs(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("DASHBOARD_LOCALS_LOG_DIR", filepath.Join(directory, "dashboard.locals"))
	files, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	files.Access.Print("method=GET path=/api/v1/health status=200")
	files.Error.Print("server error")
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}

	access, err := os.ReadFile(filepath.Join(files.Directory, "access.log"))
	if err != nil {
		t.Fatal(err)
	}
	errorLog, err := os.ReadFile(filepath.Join(files.Directory, "error.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(access), "path=/api/v1/health") || !strings.Contains(string(errorLog), "server error") {
		t.Fatalf("unexpected log contents: access=%q error=%q", access, errorLog)
	}
}
