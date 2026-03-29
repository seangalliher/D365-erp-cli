// Package batch implements JSONL batch/pipeline mode for the D365 CLI.
// It reads newline-delimited JSON commands from stdin and executes them
// sequentially, writing results to stdout as JSONL. This enables AI agents
// to pipeline multiple operations in a single CLI invocation.
package batch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

// Command represents a single command in a batch.
type Command struct {
	ID      string                 `json:"id,omitempty"`
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// Result represents the output of a single batch command.
type Result struct {
	ID       string      `json:"id,omitempty"`
	Success  bool        `json:"success"`
	Command  string      `json:"command"`
	Data     interface{} `json:"data,omitempty"`
	Error    *ErrorInfo  `json:"error,omitempty"`
	Duration int64       `json:"duration_ms"`
}

// ErrorInfo carries error details in batch results.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Executor processes batch commands.
type Executor struct {
	handler   func(cmd *Command) (*types.Response, error)
	writer    io.Writer
	stopOnErr bool
}

// NewExecutor creates a batch executor.
func NewExecutor(handler func(cmd *Command) (*types.Response, error), writer io.Writer, stopOnErr bool) *Executor {
	return &Executor{
		handler:   handler,
		writer:    writer,
		stopOnErr: stopOnErr,
	}
}

// ProcessStream reads JSONL commands from reader and writes results to writer.
func (e *Executor) ProcessStream(reader io.Reader) (*Summary, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	summary := &Summary{}
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		var cmd Command
		if err := json.Unmarshal(line, &cmd); err != nil {
			result := &Result{
				Success:  false,
				Command:  "parse",
				Error:    &ErrorInfo{Code: "PARSE_ERROR", Message: fmt.Sprintf("line %d: %v", lineNum, err)},
				Duration: 0,
			}
			summary.Errors++
			summary.Total++
			if err := e.writeResult(result); err != nil {
				return summary, err
			}
			if e.stopOnErr {
				return summary, fmt.Errorf("parse error on line %d", lineNum)
			}
			continue
		}

		start := time.Now()
		resp, err := e.handler(&cmd)
		duration := time.Since(start).Milliseconds()

		var result *Result
		if err != nil {
			result = &Result{
				ID:       cmd.ID,
				Success:  false,
				Command:  cmd.Command,
				Error:    &ErrorInfo{Code: "EXEC_ERROR", Message: err.Error()},
				Duration: duration,
			}
			summary.Errors++
		} else if resp != nil && !resp.Success {
			errInfo := &ErrorInfo{}
			if resp.Error != nil {
				errInfo.Code = resp.Error.Code
				errInfo.Message = resp.Error.Message
			}
			result = &Result{
				ID:       cmd.ID,
				Success:  false,
				Command:  cmd.Command,
				Error:    errInfo,
				Duration: duration,
			}
			summary.Errors++
		} else {
			var data interface{}
			if resp != nil {
				data = resp.Data
			}
			result = &Result{
				ID:       cmd.ID,
				Success:  true,
				Command:  cmd.Command,
				Data:     data,
				Duration: duration,
			}
			summary.Succeeded++
		}

		summary.Total++
		if err := e.writeResult(result); err != nil {
			return summary, err
		}

		if !result.Success && e.stopOnErr {
			return summary, fmt.Errorf("command %q failed", cmd.Command)
		}
	}

	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("reading batch input: %w", err)
	}

	return summary, nil
}

func (e *Executor) writeResult(result *Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	if _, err := e.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write result: %w", err)
	}
	return nil
}

// Summary contains aggregate results from a batch run.
type Summary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Errors    int `json:"errors"`
}
