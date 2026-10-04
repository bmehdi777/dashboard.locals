package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type statsFlags struct {
	from        string
	to          string
	project     string
	timezone    string
	tools       string
	granularity string
}

func newStatsCommand(state *commandState) *cobra.Command {
	stats := &cobra.Command{
		Use:   "stats",
		Short: "Inspect and synchronize usage statistics",
		Args:  noArgs,
	}
	stats.AddCommand(newStatsShowCommand(state))
	stats.AddCommand(newStatsSyncCommand(state))
	stats.AddCommand(newStatsCompactCommand(state))
	return stats
}

func (f *statsFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&f.from, "from", "", "start of the RFC3339 time range")
	command.Flags().StringVar(&f.to, "to", "", "end of the RFC3339 time range")
	command.Flags().StringVar(&f.project, "project", "", "OpenCode project filter")
	command.Flags().StringVar(&f.timezone, "timezone", "", "statistics timezone")
	command.Flags().StringVar(&f.tools, "tools", "", "tool detail level: none, summary or detail")
	command.Flags().StringVar(&f.granularity, "granularity", "", "statistics granularity: daily or monthly")
}

func (f statsFlags) normalized() (statsRequest, error) {
	from, err := normalizeOptionalTime(f.from, "from")
	if err != nil {
		return statsRequest{}, err
	}
	to, err := normalizeOptionalTime(f.to, "to")
	if err != nil {
		return statsRequest{}, err
	}
	return statsRequest{
		From: from, To: to, Project: strings.TrimSpace(f.project),
		Timezone: strings.TrimSpace(f.timezone), Tools: strings.TrimSpace(f.tools),
		Granularity: strings.TrimSpace(f.granularity),
	}, nil
}

func (f statsFlags) query() (url.Values, error) {
	request, err := f.normalized()
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	if request.From != "" {
		query.Set("from", request.From)
	}
	if request.To != "" {
		query.Set("to", request.To)
	}
	if request.Project != "" {
		query.Set("project", request.Project)
	}
	if request.Timezone != "" {
		query.Set("timezone", request.Timezone)
	}
	if request.Tools != "" {
		query.Set("tools", request.Tools)
	}
	if request.Granularity != "" {
		query.Set("granularity", request.Granularity)
	}
	return query, nil
}

func newStatsShowCommand(state *commandState) *cobra.Command {
	var flags statsFlags
	command := &cobra.Command{
		Use:   "show",
		Short: "Display stored statistics",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			query, err := flags.query()
			if err != nil {
				return err
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodGet, "/api/v1/stats", query, nil, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response statsView
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: statistics response is invalid", ErrInvalidResponse)
			}
			_, _ = fmt.Fprintf(state.out, "source: %s\nrange: %s - %s\ngranularity: %s\nsync: %s\naggregates: %d\n", response.Source, formatTime(response.From), formatTime(response.To), response.Granularity, response.Sync.Status, len(response.Aggregates))
			for _, aggregate := range response.Aggregates {
				_, _ = fmt.Fprintf(state.out, "%s - %s: sessions=%d prompts=%d steps=%d input=%d output=%d cost=%.6f\n", formatTime(aggregate.PeriodStart), formatTime(aggregate.PeriodEnd), aggregate.Sessions, aggregate.Prompts, aggregate.Steps, aggregate.InputTokens, aggregate.OutputTokens, aggregate.Cost)
			}
			return nil
		},
	}
	flags.addFlags(command)
	return command
}

func newStatsSyncCommand(state *commandState) *cobra.Command {
	var flags statsFlags
	command := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize OpenCode statistics through the daemon",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			request, err := flags.normalized()
			if err != nil {
				return err
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPost, "/api/v1/stats/sync", nil, request, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response statsSyncResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: synchronization response is invalid", ErrInvalidResponse)
			}
			_, _ = fmt.Fprintf(state.out, "status: %s\nsource: %s\ndetection: %s\nraw: %d created, %d existing\naggregates: %d created, %d updated\nrange: %s - %s\n", response.Status, response.Source, response.Detection.State, response.RawCreated, response.RawExisting, response.AggregatesCreated, response.AggregatesUpdated, formatTime(response.From), formatTime(response.To))
			return nil
		},
	}
	flags.addFlags(command)
	return command
}

func newStatsCompactCommand(state *commandState) *cobra.Command {
	var before, project, granularity string
	var dryRun, dropRaw bool
	command := &cobra.Command{
		Use:   "compact --before <timestamp>",
		Short: "Compact raw statistics into aggregates",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			before, err := normalizeRequiredTime(before, "before")
			if err != nil {
				return err
			}
			request := compactStatsRequest{
				Before: before, Project: strings.TrimSpace(project),
				Granularity: strings.TrimSpace(granularity), DryRun: dryRun, DropRaw: dropRaw,
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPost, "/api/v1/stats/compact", nil, request, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response compactStatsResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: compaction response is invalid", ErrInvalidResponse)
			}
			_, _ = fmt.Fprintf(state.out, "source: %s\ngranularity: %s\ndry-run: %t\nraw read: %d\naggregates: %d created, %d updated\nraw deleted: %d\n", response.Source, response.Granularity, response.DryRun, response.RawRead, response.AggregatesCreated, response.AggregatesUpdated, response.RawDeleted)
			return nil
		},
	}
	command.Flags().StringVar(&before, "before", "", "compact records ending before this RFC3339 timestamp")
	command.Flags().StringVar(&project, "project", "", "OpenCode project filter")
	command.Flags().StringVar(&granularity, "granularity", "", "target granularity: daily or monthly")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "calculate changes without writing or deleting data")
	command.Flags().BoolVar(&dropRaw, "drop-raw", false, "delete compacted raw records explicitly")
	return command
}

func normalizeOptionalTime(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: --%s must be an RFC3339 timestamp", ErrUsage, name)
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func normalizeRequiredTime(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: --%s is required", ErrUsage, name)
	}
	return normalizeOptionalTime(value, name)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
