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

func newSearchCommand(state *commandState) *cobra.Command {
	searchCommand := &cobra.Command{
		Use:   "search",
		Short: "Manage search roots and search source files",
		Args:  noArgs,
	}
	searchCommand.AddCommand(newSearchRootsCommand(state))
	searchCommand.AddCommand(newSearchRunCommand(state))
	searchCommand.AddCommand(newSearchOpenCommand(state))
	return searchCommand
}

func newSearchRootsCommand(state *commandState) *cobra.Command {
	roots := &cobra.Command{
		Use:   "roots",
		Short: "Manage search roots",
		Args:  noArgs,
	}

	var includeDisabled bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List configured search roots",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			query := url.Values{}
			if includeDisabled {
				query.Set("includeDisabled", "true")
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodGet, "/api/v1/search-roots", query, nil, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response rootsResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: search roots response is invalid", ErrInvalidResponse)
			}
			for _, root := range response.Data {
				status := "disabled"
				if root.Enabled {
					status = "enabled"
				}
				_, _ = fmt.Fprintf(state.out, "%s\t%s\t%s\t%s\n", root.ID, status, root.Name, root.Path)
			}
			return nil
		},
	}
	list.Flags().BoolVar(&includeDisabled, "include-disabled", false, "include disabled roots")

	var disabled bool
	create := &cobra.Command{
		Use:   "create <name> <path>",
		Short: "Create a search root",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled := !disabled
			request := searchRootRequest{Name: args[0], Path: args[1], Enabled: &enabled}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPost, "/api/v1/search-roots", nil, request, &raw); err != nil {
				return err
			}
			return state.renderRoot(raw, "created")
		},
	}
	create.Flags().BoolVar(&disabled, "disabled", false, "create the root disabled")

	var name, path string
	var enabled, updateDisabled bool
	update := &cobra.Command{
		Use:   "update <root-id>",
		Short: "Update a search root",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if !flags.Changed("name") && !flags.Changed("path") && !flags.Changed("enabled") && !flags.Changed("disabled") {
				return fmt.Errorf("%w: at least one update flag is required", ErrUsage)
			}
			if flags.Changed("enabled") && flags.Changed("disabled") {
				return fmt.Errorf("%w: --enabled and --disabled are mutually exclusive", ErrUsage)
			}
			request := updateSearchRootRequest{}
			if flags.Changed("name") {
				request.Name = &name
			}
			if flags.Changed("path") {
				request.Path = &path
			}
			if flags.Changed("enabled") {
				request.Enabled = &enabled
			}
			if flags.Changed("disabled") {
				value := !updateDisabled
				request.Enabled = &value
			}
			var raw json.RawMessage
			endpoint := "/api/v1/search-roots/" + url.PathEscape(args[0])
			if err := state.do(cmd.Context(), http.MethodPatch, endpoint, nil, request, &raw); err != nil {
				return err
			}
			return state.renderRoot(raw, "updated")
		},
	}
	update.Flags().StringVar(&name, "name", "", "new root name")
	update.Flags().StringVar(&path, "path", "", "new root path")
	update.Flags().BoolVar(&enabled, "enabled", false, "enable the root")
	update.Flags().BoolVar(&updateDisabled, "disabled", false, "disable the root")

	deleteCommand := &cobra.Command{
		Use:   "delete <root-id>",
		Short: "Delete a search root",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint := "/api/v1/search-roots/" + url.PathEscape(args[0])
			if err := state.do(cmd.Context(), http.MethodDelete, endpoint, nil, nil, nil); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderJSON(map[string]string{"status": "deleted", "id": args[0]})
			}
			_, _ = fmt.Fprintf(state.out, "deleted: %s\n", args[0])
			return nil
		},
	}

	roots.AddCommand(list, create, update, deleteCommand)
	return roots
}

func (s *commandState) renderRoot(raw json.RawMessage, verb string) error {
	if s.jsonOutput {
		return s.renderRaw(raw)
	}
	var root searchRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("%w: search root response is invalid", ErrInvalidResponse)
	}
	status := "disabled"
	if root.Enabled {
		status = "enabled"
	}
	_, _ = fmt.Fprintf(s.out, "%s: %s (%s)\n", verb, root.Name, status)
	_, _ = fmt.Fprintf(s.out, "id: %s\npath: %s\n", root.ID, root.Path)
	return nil
}

func newSearchRunCommand(state *commandState) *cobra.Command {
	var rootID string
	var literal, noGitignore, includeBinary bool
	var maxResults int
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "run <query>",
		Short: "Search a configured root with ripgrep through the daemon",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootID, err := requiredString(rootID, "root")
			if err != nil {
				return err
			}
			if maxResults < 0 {
				return fmt.Errorf("%w: --max-results cannot be negative", ErrUsage)
			}
			if timeout < 0 || timeout > 5*time.Minute {
				return fmt.Errorf("%w: --search-timeout must be between 1ms and 5m", ErrUsage)
			}
			request := searchRequest{RootID: rootID, Query: args[0]}
			flags := cmd.Flags()
			if flags.Changed("literal") {
				request.Literal = &literal
			}
			if flags.Changed("no-gitignore") {
				value := !noGitignore
				request.RespectGitignore = &value
			}
			if flags.Changed("include-binary") {
				request.IncludeBinary = &includeBinary
			}
			if flags.Changed("max-results") {
				request.MaxResults = &maxResults
			}
			if flags.Changed("search-timeout") {
				milliseconds := int(timeout / time.Millisecond)
				if milliseconds < 1 {
					milliseconds = 1
				}
				request.TimeoutMS = &milliseconds
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPost, "/api/v1/search", nil, request, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response searchResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: search response is invalid", ErrInvalidResponse)
			}
			for _, result := range response.Results {
				_, _ = fmt.Fprintf(state.out, "%s:%d:%d: %s\n", result.Path, result.Line, result.Column, strings.TrimSpace(result.Excerpt))
			}
			_, _ = fmt.Fprintf(state.out, "%d result(s)\n", response.Count)
			return nil
		},
	}
	command.Flags().StringVar(&rootID, "root", "", "search root ID")
	command.Flags().BoolVar(&literal, "literal", false, "search the query as a literal string")
	command.Flags().BoolVar(&noGitignore, "no-gitignore", false, "do not respect .gitignore files")
	command.Flags().BoolVar(&includeBinary, "include-binary", false, "include binary files")
	command.Flags().IntVar(&maxResults, "max-results", 0, "maximum number of results")
	command.Flags().DurationVar(&timeout, "search-timeout", 0, "search execution timeout")
	return command
}

func newSearchOpenCommand(state *commandState) *cobra.Command {
	var rootID string
	var line, column int
	command := &cobra.Command{
		Use:   "open <path>",
		Short: "Open a search-root file in the configured editor",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootID, err := requiredString(rootID, "root")
			if err != nil {
				return err
			}
			if line < 1 || column < 1 {
				return fmt.Errorf("%w: --line and --column must be positive", ErrUsage)
			}
			request := openFileRequest{RootID: rootID, Path: args[0], Line: line, Column: column}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPost, "/api/v1/files/open", nil, request, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response openFileResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: open response is invalid", ErrInvalidResponse)
			}
			_, _ = fmt.Fprintf(state.out, "%s\n", response.Status)
			return nil
		},
	}
	command.Flags().StringVar(&rootID, "root", "", "search root ID")
	command.Flags().IntVar(&line, "line", 1, "line number")
	command.Flags().IntVar(&column, "column", 1, "column number")
	return command
}
