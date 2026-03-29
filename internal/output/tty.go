package output

import "os"

// IsTerminal reports whether the given file is connected to a terminal (TTY).
// This is used for auto-detecting output format: JSON for pipes, table for terminals.
func IsTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
