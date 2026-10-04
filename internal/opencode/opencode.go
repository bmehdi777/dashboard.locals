package opencode

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	Source        = "opencode"
	FormatVersion = 1
)

var (
	ErrUnavailable     = errors.New("opencode is unavailable")
	ErrUnauthenticated = errors.New("opencode authentication failed")
	ErrIncompatible    = errors.New("opencode instance is incompatible")
	ErrInvalidResponse = errors.New("invalid opencode response")
)

type State string

const (
	StateAvailable       State = "available"
	StateAbsent          State = "absent"
	StateStopped         State = "stopped"
	StateUnauthenticated State = "unauthenticated"
	StateIncompatible    State = "incompatible"
)

type Detection struct {
	State               State  `json:"state"`
	Version             string `json:"version,omitempty"`
	ExecutableAvailable bool   `json:"executableAvailable"`
}

type ServerInfo struct {
	Version string
}

type ServiceFile struct {
	URL           string
	Host          string
	Port          int
	Username      string
	Token         string
	Password      string
	Authorization string
}

type Connection struct {
	BaseURL       string
	authorization string
}

type Detector struct {
	ServiceFile string
	Enabled     bool
	HTTPClient  *http.Client
	LookPath    func(string) (string, error)
}

func NewDetector(serviceFile string, timeout time.Duration) *Detector {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Detector{
		ServiceFile: serviceFile,
		Enabled:     true,
		HTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
		LookPath: exec.LookPath,
	}
}

func (d *Detector) Detect(ctx context.Context) Detection {
	if !d.Enabled {
		return Detection{State: StateAbsent}
	}
	executableAvailable := false
	if d.LookPath != nil {
		_, err := d.LookPath("opencode")
		executableAvailable = err == nil
	}
	servicePath := d.ServiceFile
	if servicePath == "" {
		home, _ := os.UserHomeDir()
		servicePath = defaultServiceFile(home)
	}
	data, err := os.ReadFile(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Detection{State: StateAbsent, ExecutableAvailable: executableAvailable}
		}
		return Detection{State: StateStopped, ExecutableAvailable: executableAvailable}
	}
	file, err := ParseServiceFile(data)
	if err != nil {
		return Detection{State: StateIncompatible, ExecutableAvailable: executableAvailable}
	}
	connection, err := connectionFromServiceFile(file)
	if err != nil {
		return Detection{State: StateIncompatible, ExecutableAvailable: executableAvailable}
	}
	client := NewClient(connection, d.HTTPClient)
	info, err := client.Info(ctx)
	if err != nil {
		state := StateStopped
		if errors.Is(err, ErrUnauthenticated) {
			state = StateUnauthenticated
		} else if errors.Is(err, ErrIncompatible) || errors.Is(err, ErrInvalidResponse) {
			state = StateIncompatible
		}
		return Detection{State: state, ExecutableAvailable: executableAvailable}
	}
	return Detection{State: StateAvailable, Version: info.Version, ExecutableAvailable: executableAvailable}
}

func (d *Detector) Connect(ctx context.Context) (*Client, Detection, error) {
	detection := d.Detect(ctx)
	if detection.State != StateAvailable {
		return nil, detection, stateError(detection.State)
	}
	servicePath := d.ServiceFile
	if servicePath == "" {
		home, _ := os.UserHomeDir()
		servicePath = defaultServiceFile(home)
	}
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, detection, ErrUnavailable
	}
	file, err := ParseServiceFile(data)
	if err != nil {
		return nil, detection, ErrIncompatible
	}
	connection, err := connectionFromServiceFile(file)
	if err != nil {
		return nil, detection, ErrIncompatible
	}
	return NewClient(connection, d.HTTPClient), detection, nil
}

func stateError(state State) error {
	switch state {
	case StateUnauthenticated:
		return ErrUnauthenticated
	case StateIncompatible:
		return ErrIncompatible
	case StateAbsent, StateStopped:
		return ErrUnavailable
	default:
		return ErrUnavailable
	}
}

func defaultServiceFile(home string) string {
	if home == "" {
		return ""
	}
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "opencode", "service.json")
}

func ParseServiceFile(data []byte) (ServiceFile, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ServiceFile{}, fmt.Errorf("%w: service file is not JSON", ErrIncompatible)
	}
	file := ServiceFile{
		URL:           stringValue(raw, "url", "baseUrl", "baseURL", "serverUrl"),
		Host:          stringValue(raw, "host", "hostname", "address"),
		Port:          intValue(raw, "port"),
		Username:      stringValue(raw, "username", "user"),
		Token:         stringValue(raw, "token", "apiKey", "api_key"),
		Password:      stringValue(raw, "password"),
		Authorization: stringValue(raw, "authorization", "auth"),
	}
	if file.URL == "" && file.Host == "" && file.Port == 0 {
		return ServiceFile{}, fmt.Errorf("%w: service file has no endpoint", ErrIncompatible)
	}
	return file, nil
}

func stringValue(raw map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var result string
			if json.Unmarshal(value, &result) == nil && strings.TrimSpace(result) != "" {
				return strings.TrimSpace(result)
			}
		}
	}
	return ""
}

func intValue(raw map[string]json.RawMessage, keys ...string) int {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var result int
			if json.Unmarshal(value, &result) == nil {
				return result
			}
			var text string
			if json.Unmarshal(value, &text) == nil {
				result, _ = strconv.Atoi(text)
				return result
			}
		}
	}
	return 0
}

func connectionFromServiceFile(file ServiceFile) (Connection, error) {
	base := strings.TrimSpace(file.URL)
	if base == "" {
		host := strings.TrimSpace(file.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		if strings.Contains(host, "://") {
			base = host
		} else {
			port := file.Port
			if port == 0 {
				port = 4096
			}
			base = "http://" + host + ":" + strconv.Itoa(port)
		}
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Connection{}, fmt.Errorf("%w: invalid service URL", ErrIncompatible)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	authorization := strings.TrimSpace(file.Authorization)
	if authorization == "" && file.Token != "" {
		authorization = "Bearer " + file.Token
	}
	if authorization == "" && file.Password != "" {
		username := file.Username
		if username == "" {
			username = "opencode"
		}
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+file.Password))
	}
	return Connection{BaseURL: strings.TrimRight(parsed.String(), "/"), authorization: authorization}, nil
}

type Client struct {
	connection Connection
	httpClient *http.Client
}

func NewClient(connection Connection, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{connection: connection, httpClient: httpClient}
}

func (c *Client) Info(ctx context.Context) (ServerInfo, error) {
	var payload struct {
		Version string `json:"version"`
	}
	if err := c.getJSON(ctx, "/api/info", nil, &payload); err != nil {
		return ServerInfo{}, err
	}
	if strings.TrimSpace(payload.Version) == "" {
		return ServerInfo{}, ErrInvalidResponse
	}
	return ServerInfo{Version: payload.Version}, nil
}

type StatsQuery struct {
	From     *time.Time
	To       *time.Time
	Project  string
	Timezone string
	Tools    string
}

type UsageSnapshot struct {
	Range      StatsRange
	Sessions   int64
	Subagents  int64
	Prompts    int64
	Steps      int64
	Tokens     TokenUsage
	Cost       float64
	Tools      json.RawMessage
	ActiveDays int64
	Streak     int64
	Activity   []Activity
	Models     []ModelUsage
	Payload    json.RawMessage
}

type StatsRange struct {
	From time.Time
	To   time.Time
}

type TokenUsage struct {
	Input      int64
	Output     int64
	Reasoning  int64
	CacheRead  int64
	CacheWrite int64
}

func (usage *TokenUsage) UnmarshalJSON(data []byte) error {
	var value struct {
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	usage.Input = value.Input
	usage.Output = value.Output
	usage.Reasoning = value.Reasoning
	usage.CacheRead = value.Cache.Read
	usage.CacheWrite = value.Cache.Write
	return nil
}

func (usage TokenUsage) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	}{
		Input: usage.Input, Output: usage.Output, Reasoning: usage.Reasoning,
		Cache: struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		}{Read: usage.CacheRead, Write: usage.CacheWrite},
	})
}

type Activity struct {
	Date  string `json:"date"`
	Steps int64  `json:"steps"`
}

type ModelUsage struct {
	Model struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
		Variant    string `json:"variant,omitempty"`
	} `json:"model"`
	Steps  int64      `json:"steps"`
	Tokens TokenUsage `json:"tokens"`
	Cost   float64    `json:"cost"`
}

func (c *Client) FetchStats(ctx context.Context, query StatsQuery) (UsageSnapshot, error) {
	values := url.Values{}
	if query.From != nil {
		values.Set("from", strconv.FormatInt(query.From.UTC().UnixMilli(), 10))
	}
	if query.To != nil {
		values.Set("to", strconv.FormatInt(query.To.UTC().UnixMilli(), 10))
	}
	if query.Project != "" {
		values.Set("project", query.Project)
	}
	if query.Timezone != "" {
		values.Set("timezone", query.Timezone)
	}
	if query.Tools != "" {
		values.Set("tools", query.Tools)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.getJSON(ctx, "/api/experimental/session/stats", values, &envelope); err != nil {
		return UsageSnapshot{}, err
	}
	if len(envelope.Data) == 0 || !json.Valid(envelope.Data) {
		return UsageSnapshot{}, ErrInvalidResponse
	}
	var payload struct {
		Range struct {
			From float64 `json:"from"`
			To   float64 `json:"to"`
		} `json:"range"`
		Sessions   int64           `json:"sessions"`
		Subagents  int64           `json:"subagents"`
		Prompts    int64           `json:"prompts"`
		Steps      int64           `json:"steps"`
		Tokens     TokenUsage      `json:"tokens"`
		Cost       float64         `json:"cost"`
		Tools      json.RawMessage `json:"tools"`
		ActiveDays int64           `json:"activeDays"`
		Streak     int64           `json:"streak"`
		Activity   []Activity      `json:"activity"`
		Models     []ModelUsage    `json:"models"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return UsageSnapshot{}, fmt.Errorf("%w: decode statistics", ErrInvalidResponse)
	}
	from := timeFromMilliseconds(payload.Range.From)
	to := timeFromMilliseconds(payload.Range.To)
	if from.IsZero() && query.From != nil {
		from = query.From.UTC()
	}
	if to.IsZero() && query.To != nil {
		to = query.To.UTC()
	}
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return UsageSnapshot{}, fmt.Errorf("%w: invalid statistics range", ErrInvalidResponse)
	}
	return UsageSnapshot{
		Range: StatsRange{From: from, To: to}, Sessions: payload.Sessions,
		Subagents: payload.Subagents, Prompts: payload.Prompts, Steps: payload.Steps,
		Tokens: payload.Tokens, Cost: payload.Cost, Tools: payload.Tools,
		ActiveDays: payload.ActiveDays, Streak: payload.Streak,
		Activity: payload.Activity, Models: payload.Models,
		Payload: append(json.RawMessage(nil), envelope.Data...),
	}, nil
}

func timeFromMilliseconds(value float64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value < 100000000000 {
		value *= 1000
	}
	return time.UnixMilli(int64(value)).UTC()
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, target any) error {
	endpoint := c.connection.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ErrUnavailable
	}
	request.Header.Set("Accept", "application/json")
	if c.connection.authorization != "" {
		request.Header.Set("Authorization", c.connection.authorization)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ErrUnavailable
		}
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrUnauthenticated
	}
	if response.StatusCode == http.StatusNotFound {
		return ErrIncompatible
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ErrUnavailable
	}
	limited := io.LimitReader(response.Body, 16<<20)
	if err := json.NewDecoder(limited).Decode(target); err != nil {
		return ErrInvalidResponse
	}
	return nil
}
