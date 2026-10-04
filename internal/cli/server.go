package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newServerCommand(state *commandState) *cobra.Command {
	server := &cobra.Command{
		Use:   "server",
		Short: "Inspect the dashboard daemon",
		Args:  noArgs,
	}
	health := &cobra.Command{
		Use:   "health",
		Short: "Check daemon and OpenCode availability",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodGet, "/api/v1/health", nil, nil, &raw); err != nil {
				return err
			}
			if state.jsonOutput {
				return state.renderRaw(raw)
			}
			var response healthResponse
			if err := json.Unmarshal(raw, &response); err != nil {
				return fmt.Errorf("%w: health response is invalid", ErrInvalidResponse)
			}
			_, _ = fmt.Fprintf(state.out, "status: %s\nversion: %s\n", response.Status, response.Version)
			if len(response.OpenCode) > 0 {
				var detection detectionResponse
				if err := json.Unmarshal(response.OpenCode, &detection); err == nil {
					_, _ = fmt.Fprintf(state.out, "opencode: %s", detection.State)
					if detection.Version != "" {
						_, _ = fmt.Fprintf(state.out, " (%s)", detection.Version)
					}
					_, _ = fmt.Fprintf(state.out, ", executable: %t\n", detection.ExecutableAvailable)
				}
			}
			return nil
		},
	}
	server.AddCommand(health)
	return server
}
