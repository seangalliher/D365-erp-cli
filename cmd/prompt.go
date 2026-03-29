package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/seangalliher/d365-erp-cli/internal/output"
)

// promptForArg prompts the user for a missing argument in interactive mode.
// Returns an error if not running in a terminal (non-interactive).
func promptForArg(name, hint string) (string, error) {
	if !output.IsTerminal(os.Stdout) {
		return "", fmt.Errorf("missing required argument: %s", name)
	}
	return promptForArgFrom(os.Stdin, os.Stderr, name, hint)
}

// promptForArgFrom is the testable version of promptForArg.
func promptForArgFrom(r io.Reader, w io.Writer, name, hint string) (string, error) {
	prompt := fmt.Sprintf("%s", name)
	if hint != "" {
		prompt += fmt.Sprintf(" (%s)", hint)
	}
	fmt.Fprintf(w, "%s: ", prompt)

	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "" {
			return "", fmt.Errorf("missing required argument: %s", name)
		}
		return value, nil
	}
	return "", fmt.Errorf("missing required argument: %s", name)
}
