package cli

import (
	"encoding/json"
	"time"
)

type healthResponse struct {
	Status   string          `json:"status"`
	Version  string          `json:"version"`
	OpenCode json.RawMessage `json:"opencode,omitempty"`
}

type detectionResponse struct {
	State               string `json:"state"`
	Version             string `json:"version,omitempty"`
	ExecutableAvailable bool   `json:"executableAvailable"`
}

type searchRoot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type rootsResponse struct {
	Data []searchRoot `json:"data"`
}

type searchRootRequest struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type updateSearchRootRequest struct {
	Name    *string `json:"name,omitempty"`
	Path    *string `json:"path,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type searchRequest struct {
	RootID           string `json:"rootId"`
	Query            string `json:"query"`
	Literal          *bool  `json:"literal,omitempty"`
	RespectGitignore *bool  `json:"respectGitignore,omitempty"`
	IncludeBinary    *bool  `json:"includeBinary,omitempty"`
	MaxResults       *int   `json:"maxResults,omitempty"`
	TimeoutMS        *int   `json:"timeoutMs,omitempty"`
}

type searchResponse struct {
	RootID  string         `json:"rootId"`
	Count   int            `json:"count"`
	Results []searchResult `json:"results"`
}

type searchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Excerpt string `json:"excerpt"`
}

type openFileRequest struct {
	RootID string `json:"rootId"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type openFileResponse struct {
	Status string `json:"status"`
}

type statsRequest struct {
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	Project     string `json:"project,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Tools       string `json:"tools,omitempty"`
	Granularity string `json:"granularity,omitempty"`
}

type statsView struct {
	Source      string           `json:"source"`
	Project     string           `json:"project,omitempty"`
	Granularity string           `json:"granularity"`
	From        time.Time        `json:"from"`
	To          time.Time        `json:"to"`
	Aggregates  []statsAggregate `json:"aggregates"`
	Sync        statsSyncView    `json:"sync"`
}

type statsAggregate struct {
	PeriodStart  time.Time       `json:"periodStart"`
	PeriodEnd    time.Time       `json:"periodEnd"`
	Sessions     int64           `json:"sessions"`
	Subagents    int64           `json:"subagents"`
	Prompts      int64           `json:"prompts"`
	Steps        int64           `json:"steps"`
	InputTokens  int64           `json:"inputTokens"`
	OutputTokens int64           `json:"outputTokens"`
	Reasoning    int64           `json:"reasoningTokens"`
	CacheRead    int64           `json:"cacheRead"`
	CacheWrite   int64           `json:"cacheWrite"`
	Cost         float64         `json:"cost"`
	Models       json.RawMessage `json:"models"`
	Tools        json.RawMessage `json:"tools"`
	Activity     json.RawMessage `json:"activity"`
	ActiveDays   int64           `json:"activeDays"`
	Streak       int64           `json:"streak"`
}

type statsSyncView struct {
	Status     string     `json:"status"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	LastFrom   *time.Time `json:"lastFrom,omitempty"`
	LastTo     *time.Time `json:"lastTo,omitempty"`
}

type statsSyncResponse struct {
	Source            string            `json:"source"`
	Status            string            `json:"status"`
	Detection         detectionResponse `json:"detection"`
	RawCreated        int               `json:"rawCreated"`
	RawExisting       int               `json:"rawExisting"`
	AggregatesCreated int               `json:"aggregatesCreated"`
	AggregatesUpdated int               `json:"aggregatesUpdated"`
	From              time.Time         `json:"from"`
	To                time.Time         `json:"to"`
}

type compactStatsRequest struct {
	Before      string `json:"before"`
	Project     string `json:"project,omitempty"`
	Granularity string `json:"granularity,omitempty"`
	DryRun      bool   `json:"dryRun"`
	DropRaw     bool   `json:"dropRaw"`
}

type compactStatsResponse struct {
	Source            string `json:"source"`
	Granularity       string `json:"granularity"`
	DryRun            bool   `json:"dryRun"`
	RawRead           int    `json:"rawRead"`
	AggregatesCreated int    `json:"aggregatesCreated"`
	AggregatesUpdated int    `json:"aggregatesUpdated"`
	RawDeleted        int64  `json:"rawDeleted"`
}
