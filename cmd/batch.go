package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/batch"
	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func init() {
	rootCmd.AddCommand(newBatchCmd())
}

func newBatchCmd() *cobra.Command {
	var stopOnError bool

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Execute commands from stdin as JSONL batch",
		Long: `Reads newline-delimited JSON commands from stdin and executes them
sequentially, writing results to stdout as JSONL (one JSON object per line).

This enables AI agents to pipeline multiple operations in a single
CLI invocation, reducing process spawn overhead.

Input format (one JSON object per line):
  {"id":"optional-id","command":"data find","args":{"entity":"Customers","query":"$top=5"}}

Output format (one JSON object per line):
  {"id":"optional-id","success":true,"command":"data find","data":...,"duration_ms":42}

Example:
  echo '{"command":"status"}' | .\d365 batch
  cat commands.jsonl | .\d365 batch --stop-on-error`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			handler := func(batchCmd *batch.Command) (*types.Response, error) {
				// Route batch commands to the appropriate handler
				return executeBatchCommand(batchCmd)
			}

			executor := batch.NewExecutor(handler, os.Stdout, stopOnError)
			summary, err := executor.ProcessStream(os.Stdin)
			if err != nil && stopOnError {
				// In stop-on-error mode, the error is expected and already written
				return nil
			}

			// If not stop-on-error, output the summary to stderr
			if summary != nil && !flagQuiet {
				RenderSuccess(cmd, map[string]interface{}{
					"batch_summary": summary,
				}, start)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "Stop processing on first error")
	return cmd
}

// executeBatchCommand routes a batch command to the appropriate CLI handler.
func executeBatchCommand(cmd *batch.Command) (*types.Response, error) {
	// This is a dispatcher that maps batch command names to actual handlers.
	// For now, return an error indicating the command needs routing.
	// A full implementation would parse cmd.Command and delegate to the
	// appropriate handler function.
	return &types.Response{
		Success: false,
		Command: cmd.Command,
		Error: &types.ErrorInfo{
			Code:    "NOT_IMPLEMENTED",
			Message: "Batch routing for '" + cmd.Command + "' will be implemented with command registry",
		},
	}, nil
}
