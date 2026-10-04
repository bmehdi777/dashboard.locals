package stats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dashboard.locals/internal/opencode"
	"dashboard.locals/internal/store"
)

const source = opencode.Source

type Connector interface {
	Connect(ctx context.Context) (*opencode.Client, opencode.Detection, error)
}

type Repository interface {
	ListAggregates(ctx context.Context, source, project, granularity string, from, to *time.Time) ([]store.Aggregate, error)
	GetSyncMetadata(ctx context.Context, source string) (store.SyncMetadata, error)
	ListRawStats(ctx context.Context, source, project string, before *time.Time) ([]store.RawStatRecord, error)
	UpsertRawStat(ctx context.Context, record store.RawStatRecord) (store.RawStatRecord, bool, error)
	UpsertAggregate(ctx context.Context, aggregate store.Aggregate) (store.Aggregate, bool, error)
	DeleteRawStats(ctx context.Context, source, project string, before time.Time) (int64, error)
	PutSyncMetadata(ctx context.Context, metadata store.SyncMetadata) error
}

type TransactionRepository interface {
	Repository
	WithTransaction(ctx context.Context, fn func(*store.Store) error) error
}

type Service struct {
	repo               TransactionRepository
	connector          Connector
	defaultTimezone    string
	defaultTools       string
	defaultGranularity string
}

func NewService(repo TransactionRepository, connector Connector, timezone, tools, granularity string) *Service {
	if timezone == "" {
		timezone = "UTC"
	}
	if tools == "" {
		tools = "summary"
	}
	if granularity == "" {
		granularity = "daily"
	}
	return &Service{
		repo: repo, connector: connector,
		defaultTimezone: timezone, defaultTools: tools, defaultGranularity: granularity,
	}
}

type Query struct {
	From        *time.Time
	To          *time.Time
	Project     string
	Granularity string
	Timezone    string
	Tools       string
}

type View struct {
	Source      string          `json:"source"`
	Project     string          `json:"project,omitempty"`
	Granularity string          `json:"granularity"`
	From        time.Time       `json:"from"`
	To          time.Time       `json:"to"`
	Aggregates  []AggregateView `json:"aggregates"`
	Sync        SyncView        `json:"sync"`
}

type AggregateView struct {
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

type SyncView struct {
	Status     string     `json:"status"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	LastFrom   *time.Time `json:"lastFrom,omitempty"`
	LastTo     *time.Time `json:"lastTo,omitempty"`
}

type SyncResult struct {
	Source            string             `json:"source"`
	Status            string             `json:"status"`
	Detection         opencode.Detection `json:"detection"`
	RawCreated        int                `json:"rawCreated"`
	RawExisting       int                `json:"rawExisting"`
	AggregatesCreated int                `json:"aggregatesCreated"`
	AggregatesUpdated int                `json:"aggregatesUpdated"`
	From              time.Time          `json:"from"`
	To                time.Time          `json:"to"`
}

type CompactRequest struct {
	Before      time.Time `json:"before"`
	Project     string    `json:"project,omitempty"`
	Granularity string    `json:"granularity"`
	DryRun      bool      `json:"dryRun"`
	DropRaw     bool      `json:"dropRaw"`
}

type CompactResult struct {
	Source            string `json:"source"`
	Granularity       string `json:"granularity"`
	DryRun            bool   `json:"dryRun"`
	RawRead           int    `json:"rawRead"`
	AggregatesCreated int    `json:"aggregatesCreated"`
	AggregatesUpdated int    `json:"aggregatesUpdated"`
	RawDeleted        int64  `json:"rawDeleted"`
}

func (s *Service) Get(ctx context.Context, query Query) (View, error) {
	normalized, err := s.normalizeQuery(query)
	if err != nil {
		return View{}, err
	}
	aggregates, err := s.repo.ListAggregates(ctx, source, normalized.Project, normalized.Granularity, normalized.From, normalized.To)
	if err != nil {
		return View{}, err
	}
	metadata, err := s.repo.GetSyncMetadata(ctx, source)
	if err != nil {
		return View{}, err
	}
	view := View{
		Source: source, Project: normalized.Project, Granularity: normalized.Granularity,
		From: valueOrTime(normalized.From), To: valueOrTime(normalized.To),
		Aggregates: make([]AggregateView, 0, len(aggregates)),
		Sync:       SyncView{Status: metadata.Status, LastSyncAt: metadata.LastSyncAt, LastFrom: metadata.LastFrom, LastTo: metadata.LastTo},
	}
	for _, aggregate := range aggregates {
		view.Aggregates = append(view.Aggregates, toAggregateView(aggregate))
	}
	return view, nil
}

func (s *Service) Sync(ctx context.Context, query Query) (SyncResult, error) {
	normalized, err := s.normalizeQuery(query)
	if err != nil {
		return SyncResult{}, err
	}
	if s.connector == nil {
		return SyncResult{}, opencode.ErrUnavailable
	}
	client, detection, err := s.connector.Connect(ctx)
	if err != nil {
		return SyncResult{Source: source, Status: string(detection.State), Detection: detection}, err
	}
	periods, err := splitPeriods(*normalized.From, *normalized.To, normalized.Granularity, normalized.Timezone)
	if err != nil {
		return SyncResult{}, err
	}
	type record struct {
		raw       store.RawStatRecord
		aggregate store.Aggregate
	}
	records := make([]record, 0, len(periods))
	for _, period := range periods {
		snapshot, fetchErr := client.FetchStats(ctx, opencode.StatsQuery{
			From: &period.From, To: &period.To, Project: normalized.Project,
			Timezone: normalized.Timezone, Tools: normalized.Tools,
		})
		if fetchErr != nil {
			return SyncResult{Source: source, Status: "error", Detection: detection}, fetchErr
		}
		// OpenCode reports the current instant as range.to for an in-progress
		// period. Persist the requested calendar bucket instead, otherwise every
		// synchronization creates a new aggregate for the same day or month.
		snapshot.Range = opencode.StatsRange{From: period.From, To: period.To}
		contentHash := sha256.Sum256(snapshot.Payload)
		records = append(records, record{
			raw: store.RawStatRecord{
				Source: source, Project: normalized.Project,
				PeriodStart: period.From, PeriodEnd: period.To,
				Granularity: "raw", FormatVersion: opencode.FormatVersion,
				ContentHash: hex.EncodeToString(contentHash[:]), Payload: snapshot.Payload,
				FetchedAt: time.Now().UTC(),
			},
			aggregate: aggregateFromSnapshot(snapshot, normalized.Project, normalized.Granularity),
		})
	}
	result := SyncResult{
		Source: source, Status: "ok", Detection: detection,
		From: *normalized.From, To: *normalized.To,
	}
	err = s.repo.WithTransaction(ctx, func(tx *store.Store) error {
		for _, item := range records {
			_, created, err := tx.UpsertRawStat(ctx, item.raw)
			if err != nil {
				return err
			}
			if created {
				result.RawCreated++
			} else {
				result.RawExisting++
			}
			_, createdAggregate, err := tx.UpsertAggregate(ctx, item.aggregate)
			if err != nil {
				return err
			}
			if createdAggregate {
				result.AggregatesCreated++
			} else {
				result.AggregatesUpdated++
			}
		}
		now := time.Now().UTC()
		return tx.PutSyncMetadata(ctx, store.SyncMetadata{
			Source: source, Status: "ok", LastSyncAt: &now,
			LastFrom: normalized.From, LastTo: normalized.To,
		})
	})
	if err != nil {
		return SyncResult{}, err
	}
	return result, nil
}

type period struct {
	From time.Time
	To   time.Time
}

func splitPeriods(from, to time.Time, granularity, timezone string) ([]period, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	localFrom := from.In(location)
	localTo := to.In(location)
	periods := make([]period, 0)
	current := time.Date(localFrom.Year(), localFrom.Month(), localFrom.Day(), 0, 0, 0, 0, location)
	for current.Before(localTo) {
		var next time.Time
		switch granularity {
		case "daily":
			next = time.Date(current.Year(), current.Month(), current.Day()+1, 0, 0, 0, 0, location)
		case "monthly":
			next = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, location)
		default:
			return nil, errors.New("granularity must be daily or monthly")
		}
		if !next.After(current) {
			return nil, errors.New("could not advance statistics period")
		}
		periods = append(periods, period{From: current.UTC(), To: next.UTC()})
		current = next
	}
	return periods, nil
}

func (s *Service) Compact(ctx context.Context, request CompactRequest) (CompactResult, error) {
	if request.Before.IsZero() {
		return CompactResult{}, errors.New("before is required")
	}
	request.Before = request.Before.UTC()
	if request.Granularity == "" {
		request.Granularity = s.defaultGranularity
	}
	if request.Granularity != "daily" && request.Granularity != "monthly" {
		return CompactResult{}, errors.New("granularity must be daily or monthly")
	}
	result := CompactResult{Source: source, Granularity: request.Granularity, DryRun: request.DryRun}
	err := s.repo.WithTransaction(ctx, func(tx *store.Store) error {
		rawRecords, err := tx.ListRawStats(ctx, source, request.Project, &request.Before)
		if err != nil {
			return err
		}
		result.RawRead = len(rawRecords)
		groups := make(map[string]store.Aggregate, len(rawRecords))
		for _, raw := range rawRecords {
			snapshot, err := snapshotFromRaw(raw)
			if err != nil {
				return fmt.Errorf("decode raw statistics: %w", err)
			}
			snapshot.Range = opencode.StatsRange{From: raw.PeriodStart, To: raw.PeriodEnd}
			aggregate := aggregateFromSnapshot(snapshot, raw.Project, request.Granularity)
			key := compactAggregateKey(aggregate)
			if previous, ok := groups[key]; ok {
				groups[key] = mergeAggregates(previous, aggregate)
			} else {
				groups[key] = aggregate
			}
		}
		if !request.DryRun {
			for _, aggregate := range groups {
				_, created, err := tx.UpsertAggregate(ctx, aggregate)
				if err != nil {
					return err
				}
				if created {
					result.AggregatesCreated++
				} else {
					result.AggregatesUpdated++
				}
			}
		}
		if request.DropRaw && !request.DryRun {
			deleted, err := tx.DeleteRawStats(ctx, source, request.Project, request.Before)
			if err != nil {
				return err
			}
			result.RawDeleted = deleted
		}
		return nil
	})
	if err != nil {
		return CompactResult{}, err
	}
	return result, nil
}

func compactAggregateKey(aggregate store.Aggregate) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", aggregate.Project, aggregate.PeriodStart.UTC().Format(time.RFC3339Nano), aggregate.PeriodEnd.UTC().Format(time.RFC3339Nano), aggregate.Granularity, aggregate.FormatVersion)
}

func mergeAggregates(left, right store.Aggregate) store.Aggregate {
	left.Sessions += right.Sessions
	left.Subagents += right.Subagents
	left.Prompts += right.Prompts
	left.Steps += right.Steps
	left.InputTokens += right.InputTokens
	left.OutputTokens += right.OutputTokens
	left.Reasoning += right.Reasoning
	left.CacheRead += right.CacheRead
	left.CacheWrite += right.CacheWrite
	left.Cost += right.Cost
	left.Models = mergeJSON(left.Models, right.Models)
	left.Tools = mergeJSON(left.Tools, right.Tools)
	left.Activity = mergeJSON(left.Activity, right.Activity)
	left.ActiveDays += right.ActiveDays
	left.Streak += right.Streak
	return left
}

func mergeJSON(left, right json.RawMessage) json.RawMessage {
	if len(left) == 0 || !json.Valid(left) {
		return right
	}
	if len(right) == 0 || !json.Valid(right) {
		return left
	}
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return right
	}
	merged := mergeJSONValues(leftValue, rightValue)
	encoded, err := json.Marshal(merged)
	if err != nil {
		return right
	}
	return encoded
}

func mergeJSONValues(left, right any) any {
	switch leftValue := left.(type) {
	case map[string]any:
		rightValue, ok := right.(map[string]any)
		if !ok {
			return right
		}
		for key, value := range rightValue {
			if previous, exists := leftValue[key]; exists {
				leftValue[key] = mergeJSONValues(previous, value)
			} else {
				leftValue[key] = value
			}
		}
		return leftValue
	case []any:
		rightValue, ok := right.([]any)
		if !ok {
			return right
		}
		return append(leftValue, rightValue...)
	case float64:
		if rightValue, ok := right.(float64); ok {
			return leftValue + rightValue
		}
	}
	return right
}

func (s *Service) normalizeQuery(query Query) (Query, error) {
	if query.Granularity == "" {
		query.Granularity = s.defaultGranularity
	}
	if query.Granularity != "daily" && query.Granularity != "monthly" {
		return Query{}, errors.New("granularity must be daily or monthly")
	}
	if query.Timezone == "" {
		query.Timezone = s.defaultTimezone
	}
	if _, err := time.LoadLocation(query.Timezone); err != nil {
		return Query{}, fmt.Errorf("invalid timezone: %w", err)
	}
	if query.Tools == "" {
		query.Tools = s.defaultTools
	}
	if query.Tools != "none" && query.Tools != "summary" && query.Tools != "detail" {
		return Query{}, errors.New("tools must be none, summary or detail")
	}
	now := time.Now().UTC()
	if query.To == nil {
		query.To = &now
	} else {
		value := query.To.UTC()
		query.To = &value
	}
	if query.From == nil {
		value := query.To.Add(-30 * 24 * time.Hour)
		query.From = &value
	} else {
		value := query.From.UTC()
		query.From = &value
	}
	if !query.To.After(*query.From) {
		return Query{}, errors.New("to must be after from")
	}
	if query.To.Sub(*query.From) > 366*24*time.Hour {
		return Query{}, errors.New("statistics range cannot exceed 366 days")
	}
	return query, nil
}

func aggregateFromSnapshot(snapshot opencode.UsageSnapshot, project, granularity string) store.Aggregate {
	models, _ := json.Marshal(snapshot.Models)
	tools := snapshot.Tools
	if len(tools) == 0 || !json.Valid(tools) {
		tools = json.RawMessage(`{}`)
	}
	return store.Aggregate{
		Source: source, Project: project,
		PeriodStart: snapshot.Range.From, PeriodEnd: snapshot.Range.To,
		Granularity: granularity, FormatVersion: opencode.FormatVersion,
		Sessions: snapshot.Sessions, Subagents: snapshot.Subagents,
		Prompts: snapshot.Prompts, Steps: snapshot.Steps,
		InputTokens: snapshot.Tokens.Input, OutputTokens: snapshot.Tokens.Output,
		Reasoning: snapshot.Tokens.Reasoning, CacheRead: snapshot.Tokens.CacheRead,
		CacheWrite: snapshot.Tokens.CacheWrite, Cost: snapshot.Cost,
		Models: models, Tools: tools, Activity: mustJSON(snapshot.Activity),
		ActiveDays: snapshot.ActiveDays, Streak: snapshot.Streak,
	}
}

func snapshotFromRaw(raw store.RawStatRecord) (opencode.UsageSnapshot, error) {
	var payload struct {
		Range struct {
			From float64 `json:"from"`
			To   float64 `json:"to"`
		} `json:"range"`
		Sessions   int64                 `json:"sessions"`
		Subagents  int64                 `json:"subagents"`
		Prompts    int64                 `json:"prompts"`
		Steps      int64                 `json:"steps"`
		Tokens     opencode.TokenUsage   `json:"tokens"`
		Cost       float64               `json:"cost"`
		Tools      json.RawMessage       `json:"tools"`
		ActiveDays int64                 `json:"activeDays"`
		Streak     int64                 `json:"streak"`
		Activity   []opencode.Activity   `json:"activity"`
		Models     []opencode.ModelUsage `json:"models"`
	}
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return opencode.UsageSnapshot{}, err
	}
	from := raw.PeriodStart
	to := raw.PeriodEnd
	if payload.Range.From > 0 {
		from = epochTime(payload.Range.From)
	}
	if payload.Range.To > 0 {
		to = epochTime(payload.Range.To)
	}
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return opencode.UsageSnapshot{}, errors.New("raw statistics range is invalid")
	}
	return opencode.UsageSnapshot{
		Range: opencode.StatsRange{From: from, To: to}, Sessions: payload.Sessions,
		Subagents: payload.Subagents, Prompts: payload.Prompts, Steps: payload.Steps,
		Tokens: payload.Tokens, Cost: payload.Cost, Tools: payload.Tools,
		ActiveDays: payload.ActiveDays, Streak: payload.Streak,
		Activity: payload.Activity, Models: payload.Models, Payload: raw.Payload,
	}, nil
}

func epochTime(value float64) time.Time {
	if value < 100000000000 {
		value *= 1000
	}
	return time.UnixMilli(int64(value)).UTC()
}

func toAggregateView(aggregate store.Aggregate) AggregateView {
	return AggregateView{
		PeriodStart: aggregate.PeriodStart, PeriodEnd: aggregate.PeriodEnd,
		Sessions: aggregate.Sessions, Subagents: aggregate.Subagents,
		Prompts: aggregate.Prompts, Steps: aggregate.Steps,
		InputTokens: aggregate.InputTokens, OutputTokens: aggregate.OutputTokens,
		Reasoning: aggregate.Reasoning, CacheRead: aggregate.CacheRead,
		CacheWrite: aggregate.CacheWrite, Cost: aggregate.Cost,
		Models: aggregate.Models, Tools: aggregate.Tools, Activity: aggregate.Activity,
		ActiveDays: aggregate.ActiveDays, Streak: aggregate.Streak,
	}
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) == 0 {
		return json.RawMessage(`[]`)
	}
	return encoded
}

func valueOrTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
