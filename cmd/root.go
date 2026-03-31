// Package cmd implements the Cobra command tree for the d365 CLI.
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/client"
	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/output"
	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

// Build-time variables injected via ldflags (Makefile / GoReleaser).
// When built with plain "go build", these remain at defaults and
// init() populates them from the Go debug build info instead.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if commit == "none" && len(s.Value) >= 7 {
					commit = s.Value[:7]
				}
			case "vcs.time":
				if date == "unknown" && s.Value != "" {
					date = s.Value
				}
			case "vcs.modified":
				if s.Value == "true" && commit != "none" {
					commit += "-dirty"
				}
			}
		}
	}
}

// Global flags
var (
	flagMu      sync.RWMutex
	flagOutput  string
	flagCompany string
	flagProfile string
	flagQuiet   bool
	flagNoColor bool
	flagVerbose bool
	flagCI      bool
	flagTimeout int
)

var rootCmd = &cobra.Command{
	Use:   "d365",
	Short: "CLI for Dynamics 365 Finance & Operations",
	Long: `d365 is a command-line interface for Dynamics 365 Finance & Operations.

It provides full access to D365 data entities (OData), server actions,
and form automation through a structured, scriptable interface.

Designed for AI agents, automation pipelines, and power users.
All commands produce structured JSON output by default when piped.

Get started:
  .\d365 connect https://your-env.operations.dynamics.com
  .\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'
  .\d365 status

PowerShell tip: Always use single quotes for --query values to prevent
$select, $filter, $top etc. from being treated as PowerShell variables.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	client.SetVersion(version)
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global persistent flags
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&flagOutput, "output", "o", "", "Output format: json, table, csv, raw (default: auto-detect)")
	pf.StringVar(&flagCompany, "company", "", "D365 legal entity / company ID (e.g., USMF)")
	pf.StringVar(&flagProfile, "profile", "", "Configuration profile to use")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential output")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose logging to stderr")
	pf.BoolVar(&flagCI, "ci", false, "CI mode: implies --output json --quiet --no-color")
	pf.IntVar(&flagTimeout, "timeout", 30, "Request timeout in seconds")

	// Register sub-command groups
	rootCmd.AddCommand(newVersionCmd())
}

func initConfig() {
	flagMu.Lock()
	defer flagMu.Unlock()

	// CI mode overrides
	if flagCI || os.Getenv(config.EnvCI) == "true" {
		flagOutput = "json"
		flagQuiet = true
		flagNoColor = true
		flagCI = true
	}

	// Environment variable defaults
	if flagCompany == "" {
		flagCompany = config.GetEnvOrDefault(config.EnvCompany, "")
	}
	if flagProfile == "" {
		flagProfile = config.GetEnvOrDefault(config.EnvProfile, "")
	}
	if flagOutput == "" {
		flagOutput = config.GetEnvOrDefault(config.EnvOutput, "")
	}
}

// GetRenderer creates a Renderer based on the current flag state.
func GetRenderer() *output.Renderer {
	flagMu.RLock()
	o, q := flagOutput, flagQuiet
	flagMu.RUnlock()
	return output.DefaultRenderer(o, q)
}

// GetCompany returns the active company from flags, env, session, or config.
func GetCompany() string {
	flagMu.RLock()
	company := flagCompany
	profile := flagProfile
	flagMu.RUnlock()

	if company != "" {
		return company
	}
	sess, err := config.LoadSession()
	if err == nil && sess.Company != "" {
		return sess.Company
	}
	cfg, err := config.Load()
	if err == nil {
		p := cfg.ActiveProfile(profile)
		if p.Company != "" {
			return p.Company
		}
	}
	return ""
}

// GetEnvironment returns the active environment URL from session or config.
func GetEnvironment() string {
	if url := os.Getenv(config.EnvURL); url != "" {
		return url
	}
	flagMu.RLock()
	profile := flagProfile
	flagMu.RUnlock()

	sess, err := config.LoadSession()
	if err == nil && sess.Environment != "" {
		return sess.Environment
	}
	cfg, err := config.Load()
	if err == nil {
		p := cfg.ActiveProfile(profile)
		return p.Environment
	}
	return ""
}

// RequireSession checks that there is an active session, returning an error if not.
func RequireSession() (*config.Session, error) {
	sess, err := config.LoadSession()
	if err != nil {
		return nil, err
	}
	if !sess.Connected {
		return nil, fmt.Errorf("no active session")
	}
	return sess, nil
}

// RenderSuccess outputs a successful response.
func RenderSuccess(cmd *cobra.Command, data interface{}, start time.Time) {
	meta := types.NewMetadata(time.Since(start))
	meta.Company = GetCompany()
	meta.Environment = GetEnvironment()
	meta.Version = version

	resp := types.SuccessResponse(cmd.CommandPath(), data, meta)
	r := GetRenderer()
	_ = r.Render(resp)
}

// RenderError outputs an error response and exits with the appropriate code.
func RenderError(cmd *cobra.Command, errInfo *types.ErrorInfo, start time.Time) {
	meta := types.NewMetadata(time.Since(start))
	meta.Company = GetCompany()
	meta.Environment = GetEnvironment()
	meta.Version = version

	resp := types.ErrorResponse(cmd.CommandPath(), errInfo, meta)
	r := GetRenderer()
	exitCode := r.RenderError(resp)
	os.Exit(exitCode)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			// Print banner to stderr in interactive mode so JSON on stdout stays clean.
			if output.IsTerminal(os.Stdout) && !flagQuiet {
				fmt.Fprint(os.Stderr, banner())
			}

			data := map[string]string{
				"version": version,
				"commit":  commit,
				"date":    date,
				"go":      goVersion(),
				"os":      osInfo(),
			}
			RenderSuccess(cmd, data, start)
			return nil
		},
	}
}

func banner() string {
	flagMu.RLock()
	noColor := flagNoColor
	flagMu.RUnlock()
	noColor = noColor || os.Getenv("NO_COLOR") != ""

	if noColor {
		return `
  ____  _____ __  ____     _____ ____  ____     ____ _     ___
 |  _ \|___ // /_| ___|   | ____|  _ \|  _ \   / ___| |   |_ _|
 | | | | |_ \ '_ \___ \   |  _| | |_) | |_) | | |   | |    | |
 | |_| |___) | (_) |__) |  | |___|  _ <|  __/  | |___| |___ | |
 |____/|____/ \___/____/   |_____|_| \_\_|      \____|_____|___|

  kubectl for Dynamics 365 Finance & Operations          ` + version + `
`
	}

	const r = "\033[0m"

	return "\n" +
		"\033[1;96m  ____  _____ __  ____     _____ ____  ____     ____ _     ___" + r + "\n" +
		"\033[1;36m |  _ \\|___ // /_| ___|   | ____|  _ \\|  _ \\   / ___| |   |_ _|" + r + "\n" +
		"\033[1;34m | | | | |_ \\ '_ \\___ \\   |  _| | |_) | |_) | | |   | |    | |" + r + "\n" +
		"\033[1;35m | |_| |___) | (_) |__) |  | |___|  _ <|  __/  | |___| |___ | |" + r + "\n" +
		"\033[1;95m |____/|____/ \\___/____/   |_____|_| \\_\\_|      \\____|_____|___|" + r + "\n" +
		"\n" +
		"\033[2m  kubectl for Dynamics 365 Finance & Operations" + r + "          \033[1;32m" + version + r + "\n\n"
}

// spinnerSuppressed returns true when the spinner should be silent.
func spinnerSuppressed() bool {
	flagMu.RLock()
	q, ci := flagQuiet, flagCI
	flagMu.RUnlock()
	return !output.IsTerminal(os.Stderr) || q || ci
}

// withSpinner wraps a blocking call with a stderr spinner.
func withSpinner(message string, fn func() error) error {
	return output.WithSpinner(spinnerSuppressed(), message, fn)
}

// newSpinner creates a spinner for multi-step operations.
func newSpinner() *output.Spinner {
	return output.NewSpinner(spinnerSuppressed())
}

func goVersion() string {
	return runtime.Version()
}

func osInfo() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
