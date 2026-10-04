package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	settingsKey = "application"

	defaultSearchTimeout    = 10 * time.Second
	defaultOpenCodeTimeout  = 5 * time.Second
	defaultHTTPReadTimeout  = 15 * time.Second
	defaultHTTPWriteTimeout = 30 * time.Second
	defaultHTTPIdleTimeout  = 60 * time.Second
)

// Runtime contains operational settings. It deliberately does not contain
// persisted application preferences.
type Runtime struct {
	ListenAddr       string
	ConfigDir        string
	DatabasePath     string
	SearchTimeout    time.Duration
	OpenCodeTimeout  time.Duration
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

// LoadRuntime resolves filesystem paths and operational overrides. Persisted
// settings are loaded separately from SQLite by Service.
func LoadRuntime() (Runtime, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Runtime{}, fmt.Errorf("resolve home directory: %w", err)
	}
	return LoadRuntimeFromEnvironment(home, os.Getenv)
}

// LoadRuntimeFromEnvironment is separated from LoadRuntime so path and
// override resolution can be tested without changing the process environment.
func LoadRuntimeFromEnvironment(home string, getenv func(string) string) (Runtime, error) {
	if strings.TrimSpace(home) == "" {
		return Runtime{}, errors.New("home directory is empty")
	}

	configHome := strings.TrimSpace(getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	} else if !filepath.IsAbs(configHome) {
		return Runtime{}, errors.New("XDG_CONFIG_HOME must be an absolute path")
	}

	configDir := filepath.Join(configHome, "dashboard.locals")
	runtimeConfig := Runtime{
		ListenAddr:       valueOr(getenv("DASHBOARD_LOCALS_LISTEN_ADDR"), "127.0.0.1:8443"),
		ConfigDir:        configDir,
		DatabasePath:     filepath.Join(configDir, "database.sqlite"),
		SearchTimeout:    defaultSearchTimeout,
		OpenCodeTimeout:  defaultOpenCodeTimeout,
		HTTPReadTimeout:  defaultHTTPReadTimeout,
		HTTPWriteTimeout: defaultHTTPWriteTimeout,
		HTTPIdleTimeout:  defaultHTTPIdleTimeout,
	}

	if value := strings.TrimSpace(getenv("DASHBOARD_LOCALS_DB")); value != "" {
		runtimeConfig.DatabasePath = value
	}
	if value := strings.TrimSpace(getenv("DASHBOARD_LOCALS_SEARCH_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Runtime{}, fmt.Errorf("invalid DASHBOARD_LOCALS_SEARCH_TIMEOUT: %q", value)
		}
		runtimeConfig.SearchTimeout = parsed
	}
	if value := strings.TrimSpace(getenv("DASHBOARD_LOCALS_OPENCODE_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Runtime{}, fmt.Errorf("invalid DASHBOARD_LOCALS_OPENCODE_TIMEOUT: %q", value)
		}
		runtimeConfig.OpenCodeTimeout = parsed
	}

	return runtimeConfig, nil
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// EnsureDirectories creates only the directories owned by the application.
func (r Runtime) EnsureDirectories() error {
	if err := os.MkdirAll(r.ConfigDir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if parent := filepath.Dir(r.DatabasePath); parent != r.ConfigDir {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return fmt.Errorf("create database directory: %w", err)
		}
	}
	return nil
}

// DefaultSettings are safe application defaults. They contain no credentials.
func DefaultSettings() Settings {
	return Settings{
		Search: SearchSettings{
			RespectGitignore: true,
			IncludeBinary:    false,
			Literal:          false,
			MaxResults:       500,
			Timeout:          defaultSearchTimeout,
		},
		Editor: EditorSettings{},
		OpenCode: OpenCodeSettings{
			ServiceFile: "",
			Enabled:     true,
		},
		Stats: StatsSettings{
			Timezone:    "UTC",
			Tools:       "summary",
			Granularity: "daily",
		},
	}
}

type Settings struct {
	Search   SearchSettings   `json:"search"`
	Editor   EditorSettings   `json:"editor"`
	OpenCode OpenCodeSettings `json:"opencode"`
	Stats    StatsSettings    `json:"stats"`
}

type SearchSettings struct {
	RespectGitignore bool          `json:"respectGitignore"`
	IncludeBinary    bool          `json:"includeBinary"`
	Literal          bool          `json:"literal"`
	MaxResults       int           `json:"maxResults"`
	Timeout          time.Duration `json:"-"`
}

// MarshalJSON keeps the public settings stable by representing Timeout as a
// duration string rather than Go's nanosecond integer representation.
func (s SearchSettings) MarshalJSON() ([]byte, error) {
	type wire struct {
		RespectGitignore bool   `json:"respectGitignore"`
		IncludeBinary    bool   `json:"includeBinary"`
		Literal          bool   `json:"literal"`
		MaxResults       int    `json:"maxResults"`
		Timeout          string `json:"timeout"`
	}
	return json.Marshal(wire{
		RespectGitignore: s.RespectGitignore,
		IncludeBinary:    s.IncludeBinary,
		Literal:          s.Literal,
		MaxResults:       s.MaxResults,
		Timeout:          s.Timeout.String(),
	})
}

func (s *SearchSettings) UnmarshalJSON(data []byte) error {
	type wire struct {
		RespectGitignore *bool  `json:"respectGitignore"`
		IncludeBinary    *bool  `json:"includeBinary"`
		Literal          *bool  `json:"literal"`
		MaxResults       *int   `json:"maxResults"`
		Timeout          string `json:"timeout"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	defaults := DefaultSettings().Search
	s.RespectGitignore = defaults.RespectGitignore
	s.IncludeBinary = defaults.IncludeBinary
	s.Literal = defaults.Literal
	s.MaxResults = defaults.MaxResults
	s.Timeout = defaults.Timeout
	if value.RespectGitignore != nil {
		s.RespectGitignore = *value.RespectGitignore
	}
	if value.IncludeBinary != nil {
		s.IncludeBinary = *value.IncludeBinary
	}
	if value.Literal != nil {
		s.Literal = *value.Literal
	}
	if value.MaxResults != nil {
		s.MaxResults = *value.MaxResults
	}
	if value.Timeout != "" {
		parsed, err := time.ParseDuration(value.Timeout)
		if err != nil {
			return fmt.Errorf("invalid search timeout: %w", err)
		}
		s.Timeout = parsed
	}
	return nil
}

type EditorSettings struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments"`
}

type OpenCodeSettings struct {
	Enabled     bool   `json:"enabled"`
	ServiceFile string `json:"serviceFile"`
}

type StatsSettings struct {
	Timezone    string `json:"timezone"`
	Tools       string `json:"tools"`
	Granularity string `json:"granularity"`
}

// SettingsRepository is the small persistence surface needed by Service.
type SettingsRepository interface {
	GetSetting(ctx context.Context, key string) ([]byte, error)
	PutSetting(ctx context.Context, key string, value []byte) error
}

type Service struct {
	repo SettingsRepository
}

func NewService(repo SettingsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	settings := DefaultSettings()
	raw, err := s.repo.GetSetting(ctx, settingsKey)
	if err != nil {
		return Settings{}, err
	}
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return Settings{}, fmt.Errorf("decode persisted settings: %w", err)
	}
	return normalizeSettings(settings)
}

func (s *Service) Save(ctx context.Context, settings Settings) (Settings, error) {
	normalized, err := normalizeSettings(settings)
	if err != nil {
		return Settings{}, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return Settings{}, fmt.Errorf("encode settings: %w", err)
	}
	if err := s.repo.PutSetting(ctx, settingsKey, raw); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}

func normalizeSettings(settings Settings) (Settings, error) {
	defaults := DefaultSettings()
	if settings.Search.MaxResults <= 0 || settings.Search.MaxResults > 10000 {
		return Settings{}, errors.New("search maxResults must be between 1 and 10000")
	}
	if settings.Search.Timeout <= 0 || settings.Search.Timeout > 5*time.Minute {
		return Settings{}, errors.New("search timeout must be between 1ns and 5m")
	}
	if strings.TrimSpace(settings.Stats.Timezone) == "" {
		settings.Stats.Timezone = defaults.Stats.Timezone
	}
	if _, err := time.LoadLocation(settings.Stats.Timezone); err != nil {
		return Settings{}, fmt.Errorf("invalid stats timezone: %w", err)
	}
	if settings.Stats.Tools != "none" && settings.Stats.Tools != "summary" && settings.Stats.Tools != "detail" {
		return Settings{}, errors.New("stats tools must be none, summary or detail")
	}
	if settings.Stats.Granularity != "daily" && settings.Stats.Granularity != "monthly" {
		return Settings{}, errors.New("stats granularity must be daily or monthly")
	}
	if settings.Editor.Command == "" && len(settings.Editor.Arguments) > 0 {
		return Settings{}, errors.New("editor command is required when arguments are configured")
	}
	if settings.OpenCode.ServiceFile != "" && !filepath.IsAbs(settings.OpenCode.ServiceFile) {
		return Settings{}, errors.New("opencode serviceFile must be absolute")
	}
	return settings, nil
}

// PublicSettings is intentionally the same shape as Settings but makes the
// API boundary explicit for future redaction changes.
func PublicSettings(settings Settings) Settings {
	settings.OpenCode.ServiceFile = sanitizePathForPublic(settings.OpenCode.ServiceFile)
	return settings
}

func sanitizePathForPublic(path string) string {
	if path == "" {
		return ""
	}
	// A configured service file is operational metadata, not a credential. Keep
	// only its basename in the public response to avoid leaking home paths.
	return filepath.Base(path)
}

func DefaultServiceFile(home string) string {
	if home == "" {
		return ""
	}
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "opencode", "service.json")
}
