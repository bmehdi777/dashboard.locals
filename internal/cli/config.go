package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

const maxConfigPatchSize = 1 << 20

func newConfigCommand(state *commandState) *cobra.Command {
	config := &cobra.Command{
		Use:   "config",
		Short: "Read and update application settings",
		Args:  noArgs,
	}

	get := &cobra.Command{
		Use:   "get",
		Short: "Display persisted settings",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodGet, "/api/v1/settings", nil, nil, &raw); err != nil {
				return err
			}
			return state.renderRaw(raw)
		},
	}

	var filename string
	patch := &cobra.Command{
		Use:   "patch --file <path|->",
		Short: "Apply a JSON settings patch through the daemon",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if filename == "" {
				return fmt.Errorf("%w: --file is required", ErrUsage)
			}
			data, err := readConfigPatch(filename, state.in)
			if err != nil {
				return err
			}
			var raw json.RawMessage
			if err := state.do(cmd.Context(), http.MethodPatch, "/api/v1/settings", nil, json.RawMessage(data), &raw); err != nil {
				return err
			}
			return state.renderRaw(raw)
		},
	}
	patch.Flags().StringVar(&filename, "file", "", "JSON patch file, or - for stdin")

	config.AddCommand(get, patch)
	return config
}

func readConfigPatch(filename string, input io.Reader) ([]byte, error) {
	var reader io.Reader = input
	var file *os.File
	if filename != "-" {
		opened, err := os.Open(filename)
		if err != nil {
			return nil, errors.New("cannot read configuration patch")
		}
		file = opened
		defer file.Close()
		reader = file
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxConfigPatchSize+1))
	if err != nil || len(data) > maxConfigPatchSize {
		return nil, errors.New("configuration patch is unreadable or too large")
	}
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, fmt.Errorf("%w: configuration patch must be a JSON object", ErrUsage)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("%w: configuration patch must contain one JSON value", ErrUsage)
	}
	return data, nil
}
