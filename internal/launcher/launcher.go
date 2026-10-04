package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"dashboard.locals/internal/search"
)

var (
	ErrNotConfigured = errors.New("editor is not configured")
	ErrInvalidEditor = errors.New("invalid editor configuration")
)

type Config struct {
	Name      string
	Command   string
	Arguments []string
}

type Runner interface {
	Start(ctx context.Context, command string, args []string) error
}

type ExecRunner struct{}

func (ExecRunner) Start(ctx context.Context, command string, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("editor executable is not available: %w", err)
	}
	// The HTTP request ends immediately after the editor is started. Using
	// CommandContext here would terminate a successfully launched editor when
	// that request context is canceled, so the detached process uses exec.Cmd
	// directly while still receiving separated, validated arguments.
	process := exec.Command(command, args...)
	if err := process.Start(); err != nil {
		return err
	}
	// Editors are intentionally detached from the request lifecycle, but the
	// process must still be waited on so it cannot become a zombie.
	go func() { _ = process.Wait() }()
	return nil
}

type Service struct {
	roots  search.RootRepository
	runner Runner
}

func NewService(roots search.RootRepository, runner Runner) *Service {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &Service{roots: roots, runner: runner}
}

func ValidateConfig(config Config) error {
	if strings.TrimSpace(config.Command) == "" {
		return ErrNotConfigured
	}
	if len(config.Arguments) > 100 {
		return fmt.Errorf("%w: too many arguments", ErrInvalidEditor)
	}
	for _, argument := range config.Arguments {
		if strings.IndexByte(argument, 0) >= 0 {
			return fmt.Errorf("%w: invalid argument", ErrInvalidEditor)
		}
		for _, placeholder := range extractPlaceholders(argument) {
			if placeholder != "file" && placeholder != "line" && placeholder != "column" {
				return fmt.Errorf("%w: unsupported placeholder", ErrInvalidEditor)
			}
		}
	}
	return nil
}

func (s *Service) Open(ctx context.Context, rootID, relativePath string, line, column int, config Config) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	if line < 1 || line > 100000000 || column < 1 || column > 100000000 {
		return fmt.Errorf("%w: line and column must be positive", ErrInvalidEditor)
	}
	root, err := s.roots.GetSearchRoot(ctx, rootID)
	if err != nil {
		return err
	}
	if !root.Enabled {
		return search.ErrRootDisabled
	}
	file, err := search.ResolveResultPath(root.Path, relativePath)
	if err != nil {
		if errors.Is(err, search.ErrSymlink) || errors.Is(err, search.ErrPathOutside) {
			return fmt.Errorf("%w: file path is not allowed", search.ErrPathOutside)
		}
		return fmt.Errorf("%w: file path is not accessible", search.ErrPathOutside)
	}
	info, err := os.Stat(file)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: target is not a regular file", search.ErrPathOutside)
	}

	values := map[string]string{
		"file":   file,
		"line":   strconv.Itoa(line),
		"column": strconv.Itoa(column),
	}
	args := make([]string, len(config.Arguments))
	for index, argument := range config.Arguments {
		args[index] = replacePlaceholders(argument, values)
	}
	return s.runner.Start(ctx, config.Command, args)
}

func extractPlaceholders(value string) []string {
	placeholders := make([]string, 0, 3)
	for _, placeholder := range []string{"file", "line", "column"} {
		if strings.Contains(value, "{"+placeholder+"}") {
			placeholders = append(placeholders, placeholder)
		}
	}
	// Detect any brace expression so unsupported placeholders cannot silently
	// pass through to an executable.
	for start := strings.IndexByte(value, '{'); start >= 0; {
		end := strings.IndexByte(value[start+1:], '}')
		if end < 0 {
			placeholders = append(placeholders, "")
			break
		}
		candidate := value[start+1 : start+1+end]
		found := false
		for _, known := range placeholders {
			if candidate == known {
				found = true
				break
			}
		}
		if !found {
			placeholders = append(placeholders, candidate)
		}
		next := start + 1 + end + 1
		if next >= len(value) {
			break
		}
		remainder := value[next:]
		nextStart := strings.IndexByte(remainder, '{')
		if nextStart < 0 {
			break
		}
		start = next + nextStart
	}
	return placeholders
}

func replacePlaceholders(value string, values map[string]string) string {
	for key, replacement := range values {
		value = strings.ReplaceAll(value, "{"+key+"}", replacement)
	}
	return value
}
