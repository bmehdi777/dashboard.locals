package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	exitOK          = 0
	exitGeneric     = 1
	exitUsage       = 2
	exitUnavailable = 3
	exitAPI         = 4
)

type RootOptions struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
	Client *Client
}

type commandState struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	client *Client

	serverURL  string
	timeout    time.Duration
	jsonOutput bool
}

// NewRootCommand creates a fresh Cobra command tree. A new tree is created
// for each invocation so tests and embedders do not share mutable flag state.
func NewRootCommand(options RootOptions) *cobra.Command {
	state := newCommandState(options)
	return state.newRootCommand()
}

func newCommandState(options RootOptions) *commandState {
	if options.In == nil {
		options.In = os.Stdin
	}
	if options.Out == nil {
		options.Out = os.Stdout
	}
	if options.ErrOut == nil {
		options.ErrOut = os.Stderr
	}
	serverURL := strings.TrimSpace(os.Getenv("DASHBOARD_LOCALS_SERVER_URL"))
	if serverURL == "" {
		serverURL = strings.TrimSpace(os.Getenv("DASHBOARD_LOCALS_URL"))
	}
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	return &commandState{
		in: options.In, out: options.Out, errOut: options.ErrOut,
		client: options.Client, serverURL: serverURL, timeout: defaultHTTPTimeout,
	}
}

func (s *commandState) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "dashboard",
		Short:         "Command-line client for dashboard.locals",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetOut(s.out)
	root.SetErr(s.errOut)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	})
	root.PersistentFlags().StringVar(&s.serverURL, "server-url", s.serverURL, "dashboard daemon URL")
	root.PersistentFlags().DurationVar(&s.timeout, "timeout", defaultHTTPTimeout, "HTTP request timeout")
	root.PersistentFlags().BoolVar(&s.jsonOutput, "json", false, "print machine-readable JSON")

	root.AddCommand(newServerCommand(s))
	root.AddCommand(newConfigCommand(s))
	root.AddCommand(newSearchCommand(s))
	root.AddCommand(newStatsCommand(s))
	return root
}

func (s *commandState) getClient() (*Client, error) {
	if s.client != nil {
		return s.client, nil
	}
	if s.timeout <= 0 {
		return nil, fmt.Errorf("%w: timeout must be positive", ErrUsage)
	}
	client, err := NewClient(s.serverURL, s.timeout)
	if err != nil {
		return nil, err
	}
	s.client = client
	return client, nil
}

func (s *commandState) do(ctx context.Context, method, endpoint string, query url.Values, payload, result any) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}
	return client.Do(ctx, method, endpoint, query, payload, result)
}

func (s *commandState) renderJSON(value any) error {
	encoder := json.NewEncoder(s.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (s *commandState) renderRaw(raw json.RawMessage) error {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, raw, "", "  "); err != nil {
		return fmt.Errorf("%w: response is not valid JSON", ErrInvalidResponse)
	}
	formatted.WriteByte('\n')
	_, err := s.out.Write(formatted.Bytes())
	return err
}

func (s *commandState) printError(err error) {
	if s.jsonOutput {
		code, message := errorDetails(err)
		_ = s.renderErrorJSON(code, message)
		return
	}
	_, _ = fmt.Fprintln(s.errOut, userErrorMessage(err))
}

func (s *commandState) renderErrorJSON(code, message string) error {
	return s.renderJSONTo(s.errOut, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func (s *commandState) renderJSONTo(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func Execute() int {
	state := newCommandState(RootOptions{})
	root := state.newRootCommand()
	if err := root.Execute(); err != nil {
		state.printError(err)
		return ExitCode(err)
	}
	return exitOK
}

func ExitCode(err error) int {
	if err == nil {
		return exitOK
	}
	if errors.Is(err, ErrUsage) {
		return exitUsage
	}
	var transportErr *TransportError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.As(err, &transportErr) {
		return exitUnavailable
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Status == 408 || apiErr.Status == 429 || apiErr.Status >= 500 {
			return exitUnavailable
		}
		return exitAPI
	}
	return exitGeneric
}

func userErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "request canceled"
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Error()
	}
	return err.Error()
}

func errorDetails(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		if code == "" {
			code = "api_error"
		}
		return code, apiErr.Message
	}
	if errors.Is(err, ErrUsage) {
		return "invalid_usage", userErrorMessage(err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "request timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled", "request canceled"
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return "daemon_unavailable", transportErr.Error()
	}
	if errors.Is(err, ErrInvalidResponse) {
		return "invalid_response", userErrorMessage(err)
	}
	return "cli_error", userErrorMessage(err)
}

func noArgs(_ *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("%w: this command does not accept positional arguments", ErrUsage)
	}
	return nil
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != count {
			return fmt.Errorf("%w: expected %d positional argument(s), got %d", ErrUsage, count, len(args))
		}
		return nil
	}
}

func requiredString(value, name string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: --%s is required", ErrUsage, name)
	}
	return value, nil
}
