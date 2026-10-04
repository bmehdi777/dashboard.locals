package search

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// StreamRunner is the line-oriented runner used by the HTTP streaming search
// endpoint. Keeping it optional preserves the small Runner interface used by
// existing callers and tests.
type StreamRunner interface {
	RunStream(ctx context.Context, dir, executable string, args []string, onLine func([]byte) error) error
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

func (ExecRunner) RunStream(ctx context.Context, dir, executable string, args []string, onLine func([]byte) error) error {
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = dir
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stdout)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if callbackErr := onLine(line); callbackErr != nil {
				_ = command.Process.Kill()
				_ = command.Wait()
				return callbackErr
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = command.Process.Kill()
			_ = command.Wait()
			return readErr
		}
	}

	runErr := command.Wait()
	if runErr == nil || isNoMatchExit(runErr) {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = "ripgrep could not complete the search"
	}
	return fmt.Errorf("%w: %s", ErrSearchExecution, sanitizeCommandError(message))
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
		if isNoMatchExit(runErr) {
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

func isNoMatchExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

type StreamResult struct {
	Count     int
	Truncated bool
}

// SearchStream emits validated matches as soon as ripgrep produces them. The
// callback is invoked in order and must return quickly so the process cannot
// outpace the HTTP client indefinitely.
func (s *Service) SearchStream(ctx context.Context, rootID string, options Options, onResult func(Result) error) (StreamResult, error) {
	if onResult == nil {
		return StreamResult{}, errors.New("search stream callback is nil")
	}
	root, err := s.roots.GetSearchRoot(ctx, rootID)
	if err != nil {
		return StreamResult{}, err
	}
	if !root.Enabled {
		return StreamResult{}, ErrRootDisabled
	}
	canonical, err := NormalizeRootPath(root.Path)
	if err != nil {
		return StreamResult{}, err
	}
	streamOptions := options
	if streamOptions.MaxResults == 0 {
		streamOptions.MaxResults = s.defaultLimit
	}
	if streamOptions.Timeout == 0 {
		streamOptions.Timeout = s.defaultTimeout
	}
	if streamOptions.Timeout <= 0 || streamOptions.Timeout > 5*time.Minute {
		return StreamResult{}, errors.New("search timeout must be between 1ns and 5m")
	}
	args, err := BuildArguments(streamOptions)
	if err != nil {
		return StreamResult{}, err
	}

	streamRunner, ok := s.runner.(StreamRunner)
	if !ok {
		results, searchErr := s.Search(ctx, rootID, streamOptions)
		if searchErr != nil {
			return StreamResult{}, searchErr
		}
		for _, result := range results {
			if callbackErr := onResult(result); callbackErr != nil {
				return StreamResult{}, callbackErr
			}
		}
		return StreamResult{Count: len(results), Truncated: len(results) >= streamOptions.MaxResults}, nil
	}

	searchContext, cancel := context.WithTimeout(ctx, streamOptions.Timeout)
	defer cancel()
	streamResult := StreamResult{}
	runErr := streamRunner.RunStream(searchContext, canonical, "rg", args, func(line []byte) error {
		result, matched, parseErr := parseResultLine(canonical, line)
		if parseErr != nil {
			return parseErr
		}
		if !matched {
			return nil
		}
		if callbackErr := onResult(result); callbackErr != nil {
			return callbackErr
		}
		streamResult.Count++
		if streamResult.Count >= streamOptions.MaxResults {
			streamResult.Truncated = true
			return ErrResultLimit
		}
		return nil
	})
	if errors.Is(runErr, ErrResultLimit) {
		return streamResult, nil
	}
	if errors.Is(searchContext.Err(), context.DeadlineExceeded) || errors.Is(runErr, context.DeadlineExceeded) {
		return StreamResult{}, ErrSearchTimeout
	}
	if runErr != nil {
		return StreamResult{}, runErr
	}
	return streamResult, nil
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
		result, matched, err := parseResultLine(root, line)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		results = append(results, result)
		if len(results) >= maxResults {
			break
		}
	}
	return results, nil
}

func parseResultLine(root string, line []byte) (Result, bool, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return Result{}, false, nil
	}
	var event rgEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return Result{}, false, fmt.Errorf("%w: invalid ripgrep output", ErrSearchExecution)
	}
	if event.Type != "match" {
		return Result{}, false, nil
	}
	path, err := decodePath(event.Data.Path)
	if err != nil {
		return Result{}, false, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
	}
	absolute, err := ResolveResultPath(root, path)
	if err != nil {
		if errors.Is(err, ErrSymlink) || errors.Is(err, ErrPathOutside) {
			return Result{}, false, fmt.Errorf("%w: unsafe result path", ErrSearchExecution)
		}
		// A file can disappear between ripgrep's scan and validation. It is
		// safer to omit it than to return a path we could not validate.
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, false, nil
		}
		return Result{}, false, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return Result{}, false, fmt.Errorf("%w: invalid result path", ErrSearchExecution)
	}
	column := 1
	if len(event.Data.Submatches) > 0 {
		column = event.Data.Submatches[0].Start + 1
	}
	excerpt := strings.TrimSuffix(event.Data.Lines.Text, "\n")
	if len(excerpt) > 16*1024 {
		excerpt = excerpt[:16*1024]
	}
	return Result{Path: filepath.ToSlash(relative), Line: event.Data.LineNumber, Column: column, Excerpt: excerpt}, true, nil
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
