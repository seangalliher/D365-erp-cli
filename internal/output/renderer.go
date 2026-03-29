// Package output provides the central rendering engine for CLI output.
// All commands route their output through the Renderer, which formats data
// according to the user's chosen format (json, table, csv, raw).
// For AI agents: non-TTY stdout auto-selects JSON.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

// Format represents an output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatCSV   Format = "csv"
	FormatRaw   Format = "raw"
)

// ParseFormat parses a format string, returning an error for invalid formats.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "json", "":
		return FormatJSON, nil
	case "table":
		return FormatTable, nil
	case "csv":
		return FormatCSV, nil
	case "raw":
		return FormatRaw, nil
	default:
		return "", fmt.Errorf("unknown output format %q: valid formats are json, table, csv, raw", s)
	}
}

// Renderer handles formatting and writing CLI output.
type Renderer struct {
	out    io.Writer
	errOut io.Writer
	format Format
	quiet  bool
}

// NewRenderer creates a renderer with the given output writers and format.
func NewRenderer(out, errOut io.Writer, format Format, quiet bool) *Renderer {
	return &Renderer{
		out:    out,
		errOut: errOut,
		format: format,
		quiet:  quiet,
	}
}

// DefaultRenderer creates a renderer using stdout/stderr with TTY auto-detection.
// If stdout is not a TTY (piped to another program), defaults to JSON.
// If stdout is a TTY (interactive terminal), defaults to table.
func DefaultRenderer(formatOverride string, quiet bool) *Renderer {
	format := FormatJSON // default for AI agents (non-TTY)
	if formatOverride != "" {
		parsed, err := ParseFormat(formatOverride)
		if err == nil {
			format = parsed
		}
	} else if IsTerminal(os.Stdout) {
		format = FormatTable // default for humans (TTY)
	}
	return NewRenderer(os.Stdout, os.Stderr, format, quiet)
}

// Render outputs a Response in the configured format.
func (r *Renderer) Render(resp *types.Response) error {
	switch r.format {
	case FormatJSON:
		return r.renderJSON(resp)
	case FormatTable:
		return r.renderTable(resp)
	case FormatCSV:
		return r.renderCSV(resp)
	case FormatRaw:
		return r.renderRaw(resp)
	default:
		return r.renderJSON(resp)
	}
}

// RenderError outputs an error response and returns the appropriate exit code.
func (r *Renderer) RenderError(resp *types.Response) int {
	_ = r.Render(resp)
	if resp.Error != nil {
		return types.ExitCodeForError(resp.Error.Code)
	}
	return types.ExitCommandError
}

func (r *Renderer) renderJSON(resp *types.Response) error {
	encoder := json.NewEncoder(r.out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(resp)
}

func (r *Renderer) renderTable(resp *types.Response) error {
	if !resp.Success && resp.Error != nil {
		fmt.Fprintf(r.errOut, "Error [%s]: %s\n", resp.Error.Code, resp.Error.Message)
		if resp.Error.Suggestion != "" {
			fmt.Fprintf(r.errOut, "Suggestion: %s\n", resp.Error.Suggestion)
		}
		return nil
	}

	if resp.Data == nil {
		if !r.quiet {
			fmt.Fprintln(r.out, "OK")
		}
		return nil
	}

	// Attempt to render data as a table
	switch data := resp.Data.(type) {
	case []interface{}:
		return r.renderSliceAsTable(data)
	case map[string]interface{}:
		return r.renderMapAsTable(data)
	default:
		// Fallback: JSON-encode the data portion
		encoder := json.NewEncoder(r.out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}
}

func (r *Renderer) renderSliceAsTable(data []interface{}) error {
	if len(data) == 0 {
		if !r.quiet {
			fmt.Fprintln(r.out, "(no results)")
		}
		return nil
	}

	// Extract headers from the first item
	firstItem, ok := data[0].(map[string]interface{})
	if !ok {
		return r.renderJSON(&types.Response{Data: data})
	}

	headers := make([]string, 0, len(firstItem))
	for k := range firstItem {
		headers = append(headers, k)
	}

	tw := NewTableWriter(r.out, headers)
	for _, item := range data {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		values := make([]string, len(headers))
		for i, h := range headers {
			values[i] = fmt.Sprintf("%v", row[h])
		}
		tw.AddRow(values)
	}
	tw.Render()
	return nil
}

func (r *Renderer) renderMapAsTable(data map[string]interface{}) error {
	tw := NewTableWriter(r.out, []string{"Field", "Value"})
	for k, v := range data {
		tw.AddRow([]string{k, fmt.Sprintf("%v", v)})
	}
	tw.Render()
	return nil
}

func (r *Renderer) renderCSV(resp *types.Response) error {
	if !resp.Success && resp.Error != nil {
		return r.renderJSON(resp) // errors always as JSON
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		return r.renderJSON(resp) // non-array data falls back to JSON
	}

	if len(data) == 0 {
		return nil
	}

	firstItem, ok := data[0].(map[string]interface{})
	if !ok {
		return r.renderJSON(resp)
	}

	headers := make([]string, 0, len(firstItem))
	for k := range firstItem {
		headers = append(headers, k)
	}

	w := csv.NewWriter(r.out)
	defer w.Flush()

	if err := w.Write(headers); err != nil {
		return err
	}

	for _, item := range data {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		values := make([]string, len(headers))
		for i, h := range headers {
			values[i] = fmt.Sprintf("%v", row[h])
		}
		if err := w.Write(values); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) renderRaw(resp *types.Response) error {
	if resp.Data == nil {
		return nil
	}
	switch data := resp.Data.(type) {
	case string:
		_, err := fmt.Fprint(r.out, data)
		return err
	default:
		encoder := json.NewEncoder(r.out)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(data)
	}
}

// Format returns the current output format.
func (r *Renderer) Format() Format {
	return r.format
}

// Out returns the output writer.
func (r *Renderer) Out() io.Writer {
	return r.out
}

// ErrOut returns the error output writer.
func (r *Renderer) ErrOut() io.Writer {
	return r.errOut
}
