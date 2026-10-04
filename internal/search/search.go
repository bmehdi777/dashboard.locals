package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidQuery    = errors.New("invalid search query")
	ErrSearchTimeout   = errors.New("search timed out")
	ErrSearchExecution = errors.New("search execution failed")
	ErrResultLimit     = errors.New("search result limit reached")
)

type Options struct {
	Query            string
	Literal          bool
	RespectGitignore bool
	IncludeBinary    bool
	MaxResults       int
	Timeout          time.Duration
}

type Result struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Excerpt string `json:"excerpt"`
}

type Runner interface {
	Run(ctx context.Context, dir, executable string, args []string) ([]byte, []byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, executable string, args []string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = dir
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

type Service struct {
	roots          RootRepository
	runner         Runner
	defaultLimit   int
	defaultTimeout time.Duration
}

func NewService(roots RootRepository, runner Runner, defaultLimit int, defaultTimeout time.Duration) *Service {
	if runner == nil {
		runner = ExecRunner{}
	}
	if defaultLimit <= 0 {
		defaultLimit = 500
	}
	if defaultTimeout <= 0 {
		defaultTimeout = 10 * time.Second
	}
	return &Service{roots: roots, runner: runner, defaultLimit: defaultLimit, defaultTimeout: defaultTimeout}
}

func BuildArguments(options Options) ([]string, error) {
	query := strings.TrimSpace(options.Query)
	if query == "" || len([]rune(query)) > 4096 {
		return nil, ErrInvalidQuery
	}
	if options.MaxResults < 0 || options.MaxResults > 10000 {
		return nil, errors.New("maxResults must be between 0 and 10000")
	}
	args := []string{"--json", "--color=never", "--no-follow"}
	if options.Literal {
		args = append(args, "--fixed-strings")
	}
	if !options.RespectGitignore {
		args = append(args, "--no-ignore")
	}
	if options.IncludeBinary {
		args = append(args, "--text")
	}
	// -- keeps a query beginning with a dash from becoming an rg option.
	return append(args, "--", query, "."), nil
}

func (s *Service) Search(ctx context.Context, rootID string, options Options) ([]Result, error) {
	root, err := s.roots.GetSearchRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if !root.Enabled {
		return nil, ErrRootDisabled
	}
	canonical, err := NormalizeRootPath(root.Path)
	if err != nil {
		return nil, err
	}
	argsOptions := options
	if argsOptions.MaxResults == 0 {
		argsOptions.MaxResults = s.defaultLimit
	}
	if argsOptions.Timeout == 0 {
		argsOptions.Timeout = s.defaultTimeout
	}
	if argsOptions.Timeout <= 0 || argsOptions.Timeout > 5*time.Minute {
		return nil, errors.New("search timeout must be between 1ns and 5m")
	}
	args, err := BuildArguments(argsOptions)
	if err != nil {
		return nil, err
	}
	searchContext, cancel := context.WithTimeout(ctx, argsOptions.Timeout)
	defer cancel()
	stdout, stderr, runErr := s.runner.Run(searchContext, canonical, "rg", args)
	if runErr != nil {
		if errors.Is(searchContext.Err(), context.DeadlineExceeded) || errors.Is(runErr, context.DeadlineExceeded) {
			return nil, ErrSearchTimeout
		}
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
			return []Result{}, nil
		}
		message := strings.TrimSpace(string(stderr))
		if message == "" {
			message = "ripgrep could not complete the search"
		}
		return nil, fmt.Errorf("%w: %s", ErrSearchExecution, sanitizeCommandError(message))
	}

	results, parseErr := parseOutput(canonical, stdout, argsOptions.MaxResults)
	if parseErr != nil {
		return nil, parseErr
	}
	return results, nil
}

func sanitizeCommandError(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}

type rgEvent struct {
	Type string `json:"type"`
	Data struct {
		Path  json.RawMessage `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
		Submatches []struct {
			Start int `json:"start"`
		} `json:"submatches"`
	} `json:"data"`
}

func parseOutput(root string, output []byte, maxResults int) ([]Result, error) {
	lines := bytes.Split(output, []byte{'\n'})
	results := make([]Result, 0)
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var event rgEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("%w: invalid ripgrep output", ErrSearchExecution)
		}
		if event.Type != "match" {
			continue
		}
		path, err := decodePath(event.Data.Path)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
		}
		absolute, err := ResolveResultPath(root, path)
		if err != nil {
			if errors.Is(err, ErrSymlink) || errors.Is(err, ErrPathOutside) {
				return nil, fmt.Errorf("%w: unsafe result path", ErrSearchExecution)
			}
			// A file can disappear between ripgrep's scan and validation. It is
			// safer to omit it than to return a path we could not validate.
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
		}
		column := 1
		if len(event.Data.Submatches) > 0 {
			column = event.Data.Submatches[0].Start + 1
		}
		excerpt := strings.TrimSuffix(event.Data.Lines.Text, "\n")
		if len(excerpt) > 16*1024 {
			excerpt = excerpt[:16*1024]
		}
		results = append(results, Result{Path: filepath.ToSlash(relative), Line: event.Data.LineNumber, Column: column, Excerpt: excerpt})
		if len(results) >= maxResults {
			break
		}
	}
	return results, nil
}

func decodePath(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var value struct {
		Text  string `json:"text"`
		Bytes string `json:"bytes"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	if value.Text != "" {
		return value.Text, nil
	}
	return value.Bytes, nil
}
