package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const DefaultDirectory = "/var/log/dashboard.locals"

type Files struct {
	Directory string
	Access    *log.Logger
	Error     *log.Logger
	warning   string
	access    *os.File
	errorLog  *os.File
}

// Open creates the access and error log files. DASHBOARD_LOCALS_LOG_DIR can be
// used for development or installations where /var/log is provisioned at a
// different location.
func Open() (*Files, error) {
	directory := strings.TrimSpace(os.Getenv("DASHBOARD_LOCALS_LOG_DIR"))
	explicitDirectory := directory != ""
	if directory == "" {
		directory = DefaultDirectory
	}

	files, err := openDirectory(directory)
	if err == nil {
		return files, nil
	}
	if explicitDirectory {
		return nil, err
	}

	// A user service normally cannot create /var/log directly. Keep the daemon
	// usable when it is launched manually and make the fallback visible in both
	// stderr (during bootstrap) and error.log.
	fallback, fallbackErr := fallbackDirectory()
	if fallbackErr != nil {
		return nil, fmt.Errorf("open logs at %s: %w; fallback failed: %v", directory, err, fallbackErr)
	}
	files, fallbackOpenErr := openDirectory(fallback)
	if fallbackOpenErr != nil {
		return nil, fmt.Errorf("open logs at %s: %w; fallback %s failed: %v", directory, err, fallback, fallbackOpenErr)
	}
	files.warning = fmt.Sprintf("unable to write logs to %s: %v; using %s", directory, err, fallback)
	return files, nil
}

func openDirectory(directory string) (*Files, error) {
	if !filepath.IsAbs(directory) {
		return nil, fmt.Errorf("log directory must be absolute: %s", directory)
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("create log directory %s: %w", directory, err)
	}
	access, err := os.OpenFile(filepath.Join(directory, "access.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open access.log: %w", err)
	}
	errorLog, err := os.OpenFile(filepath.Join(directory, "error.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		_ = access.Close()
		return nil, fmt.Errorf("open error.log: %w", err)
	}
	return &Files{
		Directory: directory,
		Access:    log.New(access, "", log.LstdFlags|log.LUTC),
		Error:     log.New(errorLog, "dashboard.locals: ", log.LstdFlags|log.LUTC),
		access:    access,
		errorLog:  errorLog,
	}, nil
}

func fallbackDirectory() (string, error) {
	stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "dashboard.locals", "log"), nil
}

func (f *Files) Warning() string {
	if f == nil {
		return ""
	}
	return f.warning
}

func (f *Files) Close() error {
	if f == nil {
		return nil
	}
	var firstErr error
	if f.access != nil {
		if err := f.access.Close(); err != nil {
			firstErr = err
		}
	}
	if f.errorLog != nil {
		if err := f.errorLog.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type discardLogger struct{}

func (discardLogger) Write(p []byte) (int, error) { return io.Discard.Write(p) }

func Discard() *log.Logger {
	return log.New(discardLogger{}, "", 0)
}
