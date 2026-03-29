// Package main is the entry point for the d365 CLI.
package main

import (
	"os"

	"github.com/seangalliher/d365-erp-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
