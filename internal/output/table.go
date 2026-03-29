package output

import (
	"fmt"
	"io"
	"strings"
)

// TableWriter renders data as an aligned ASCII table.
// It auto-calculates column widths for clean display.
type TableWriter struct {
	out     io.Writer
	headers []string
	rows    [][]string
}

// NewTableWriter creates a table writer with the given headers.
func NewTableWriter(out io.Writer, headers []string) *TableWriter {
	return &TableWriter{
		out:     out,
		headers: headers,
	}
}

// AddRow adds a row of values (must match header count).
func (tw *TableWriter) AddRow(values []string) {
	tw.rows = append(tw.rows, values)
}

// Render writes the table to the output writer.
func (tw *TableWriter) Render() {
	if len(tw.headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(tw.headers))
	for i, h := range tw.headers {
		widths[i] = len(h)
	}
	for _, row := range tw.rows {
		for i, val := range row {
			if i < len(widths) && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Cap column widths at 50 characters
	for i := range widths {
		if widths[i] > 50 {
			widths[i] = 50
		}
	}

	// Print header
	tw.printRow(tw.headers, widths)
	tw.printSeparator(widths)

	// Print rows
	for _, row := range tw.rows {
		tw.printRow(row, widths)
	}

	// Print count
	fmt.Fprintf(tw.out, "(%d rows)\n", len(tw.rows))
}

func (tw *TableWriter) printRow(values []string, widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		if len(val) > w {
			val = val[:w-3] + "..."
		}
		parts[i] = fmt.Sprintf("%-*s", w, val)
	}
	fmt.Fprintf(tw.out, "  %s\n", strings.Join(parts, "  "))
}

func (tw *TableWriter) printSeparator(widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w)
	}
	fmt.Fprintf(tw.out, "  %s\n", strings.Join(parts, "  "))
}
