package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultServerURL   = "http://127.0.0.1:8443"
	defaultHTTPTimeout = 30 * time.Second
	maxResponseSize    = 16 << 20
)

var (
	// ErrUsage marks an error caused by invalid command arguments or flags.
	ErrUsage = errors.New("invalid command usage")
	// ErrInvalidResponse marks a successful HTTP response that does not follow
	// the daemon API contract.
	ErrInvalidResponse = errors.New("invalid daemon response")
)

// APIError is the public error envelope returned by the daemon.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "daemon request failed"
	}
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// TransportError prevents callers from having to depend on the details of the
// HTTP implementation while retaining the original cause for errors.Is.
type TransportError struct {
	Err error
}

func (e *TransportError) Error() string {
	return "could not reach the dashboard daemon"
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Client is the HTTP-only client used by the CLI. It intentionally exposes no
// database or server implementation details.
type Client struct {
	baseURL         *url.URL
	httpClient      *http.Client
	maxResponseSize int64
}

// NewClient validates the daemon URL and creates a client with a bounded
// request timeout and response size.
func NewClient(rawURL string, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return newClient(rawURL, &http.Client{Timeout: timeout})
}

// NewClientWithHTTP is useful for tests and embedders that need to provide a
// custom transport. The request context remains authoritative for timeouts.
func NewClientWithHTTP(rawURL string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return newClient(rawURL, httpClient)
}

func newClient(rawURL string, httpClient *http.Client) (*Client, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("%w: server URL is required", ErrUsage)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: server URL must include an http or https host", ErrUsage)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: server URL must use http or https", ErrUsage)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%w: server URL must not contain credentials", ErrUsage)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: server URL must not contain a query or fragment", ErrUsage)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return &Client{baseURL: parsed, httpClient: httpClient, maxResponseSize: maxResponseSize}, nil
}

// Do sends one API request and decodes a JSON response into result. Endpoint
// paths are deliberately constrained to absolute API paths so a command
// cannot accidentally call an unrelated endpoint.
func (c *Client) Do(ctx context.Context, method, endpoint string, query url.Values, payload any, result any) error {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return errors.New("dashboard client is not configured")
	}
	if !strings.HasPrefix(endpoint, "/api/v1/") && endpoint != "/api/v1" {
		return fmt.Errorf("%w: invalid API endpoint", ErrUsage)
	}

	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + endpoint
	requestURL.RawPath = ""
	requestURL.RawQuery = query.Encode()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return fmt.Errorf("create daemon request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return &TransportError{Err: err}
	}
	defer response.Body.Close()

	limit := c.maxResponseSize
	if limit <= 0 {
		limit = maxResponseSize
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return fmt.Errorf("read daemon response: %w", err)
	}
	if int64(len(data)) > limit {
		return fmt.Errorf("%w: response is too large", ErrInvalidResponse)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response.StatusCode, data)
	}
	if result == nil || response.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("%w: response is not valid JSON", ErrInvalidResponse)
	}
	return nil
}

func decodeAPIError(status int, data []byte) error {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &envelope) == nil && strings.TrimSpace(envelope.Error.Message) != "" {
		return &APIError{Status: status, Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	message := http.StatusText(status)
	if message == "" {
		message = "request failed"
	}
	return &APIError{Status: status, Message: message}
}
