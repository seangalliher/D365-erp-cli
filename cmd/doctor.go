package cmd

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/auth"
	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/daemon"
)

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}

// checkResult describes the outcome of a single diagnostic check.
type checkResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "pass", "warn", "fail"
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func pass(name, msg string) checkResult { return checkResult{Name: name, Status: "pass", Message: msg} }
func warn(name, msg, sug string) checkResult {
	return checkResult{Name: name, Status: "warn", Message: msg, Suggestion: sug}
}
func fail(name, msg, sug string) checkResult {
	return checkResult{Name: name, Status: "fail", Message: msg, Suggestion: sug}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common configuration and connectivity issues",
		Long: `Run a series of diagnostic checks to verify that the CLI is properly
configured, authenticated, and can reach your D365 environment.

Checks performed:
  1. Config directory exists and is writable
  2. Config and session files parse correctly
  3. Active session is established
  4. DNS resolution for environment hostname
  5. TCP connectivity to environment (port 443)
  6. Auth tooling is available (az CLI for az-cli auth)
  7. Auth profile settings are complete (e.g., client secret env var)
  8. Token expiry status
  9. Daemon status

Example:
  .\d365 doctor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			results := runDoctorChecks()
			RenderSuccess(cmd, map[string]interface{}{
				"checks": results,
			}, start)
			return nil
		},
	}
}

func runDoctorChecks() []checkResult {
	var results []checkResult

	results = append(results, checkConfigDir())
	results = append(results, checkConfigFiles()...)
	sess := checkSessionResult(&results)
	if sess != nil && sess.Environment != "" {
		results = append(results, checkDNS(sess.Environment))
		results = append(results, checkTCPReach(sess.Environment))
	}
	results = append(results, checkAuthConfig())
	results = append(results, checkAuthProfile())
	if sess != nil {
		results = append(results, checkTokenExpiry(sess))
	}
	results = append(results, checkDaemonStatus())

	return results
}

func checkConfigDir() checkResult {
	dir, err := config.ConfigDirPath()
	if err != nil {
		return fail("config_dir", fmt.Sprintf("Cannot determine config directory: %v", err),
			"Ensure your home directory is accessible.")
	}
	// Try to ensure directory exists (also tests writability).
	_, err = config.EnsureConfigDir()
	if err != nil {
		return fail("config_dir", fmt.Sprintf("Cannot create/access config directory %s: %v", dir, err),
			"Check directory permissions.")
	}
	return pass("config_dir", fmt.Sprintf("Config directory OK: %s", dir))
}

func checkConfigFiles() []checkResult {
	var results []checkResult

	_, err := config.Load()
	if err != nil {
		results = append(results, fail("config_file", fmt.Sprintf("Config file error: %v", err),
			"Delete or fix ~/.d365cli/config.json and reconnect."))
	} else {
		results = append(results, pass("config_file", "Config file OK"))
	}

	_, err = config.LoadSession()
	if err != nil {
		results = append(results, fail("session_file", fmt.Sprintf("Session file error: %v", err),
			"Run '.\\d365 connect <url>' to create a new session."))
	} else {
		results = append(results, pass("session_file", "Session file OK"))
	}

	return results
}

func checkSessionResult(results *[]checkResult) *config.Session {
	sess, err := config.LoadSession()
	if err != nil || !sess.Connected {
		*results = append(*results, warn("session", "No active session",
			"Run '.\\d365 connect <url>' to connect to a D365 environment."))
		return nil
	}
	*results = append(*results, pass("session", fmt.Sprintf("Connected to %s", sess.Environment)))
	return sess
}

func checkDNS(envURL string) checkResult {
	u, err := url.Parse(envURL)
	if err != nil {
		return fail("dns", fmt.Sprintf("Cannot parse environment URL: %v", err),
			"Check your environment URL format.")
	}
	host := u.Hostname()
	_, err = net.LookupHost(host)
	if err != nil {
		return fail("dns", fmt.Sprintf("DNS resolution failed for %s: %v", host, err),
			"Check your network connection and DNS settings.")
	}
	return pass("dns", fmt.Sprintf("DNS resolves: %s", host))
}

func checkTCPReach(envURL string) checkResult {
	u, err := url.Parse(envURL)
	if err != nil {
		return fail("tcp", fmt.Sprintf("Cannot parse URL: %v", err), "")
	}
	host := u.Hostname()
	addr := net.JoinHostPort(host, "443")
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fail("tcp", fmt.Sprintf("Cannot reach %s: %v", addr, err),
			"Check firewall rules and network connectivity.")
	}
	conn.Close()
	return pass("tcp", fmt.Sprintf("TCP connection OK: %s", addr))
}

func checkAuthConfig() checkResult {
	_, err := exec.LookPath("az")
	if err != nil {
		return warn("auth", "Azure CLI (az) not found on PATH",
			"Install Azure CLI for az-cli auth, or use client-credentials/managed-identity auth methods.")
	}
	return pass("auth", "Azure CLI found on PATH")
}

func checkTokenExpiry(sess *config.Session) checkResult {
	if sess.TokenExpiry == "" {
		return warn("token", "No token expiry information available",
			"Run '.\\d365 connect <url>' to refresh credentials.")
	}
	expiry, err := time.Parse(time.RFC3339, sess.TokenExpiry)
	if err != nil {
		return warn("token", fmt.Sprintf("Cannot parse token expiry: %v", err),
			"Run '.\\d365 connect <url>' to refresh credentials.")
	}
	if time.Now().After(expiry) {
		return fail("token", fmt.Sprintf("Token expired at %s", sess.TokenExpiry),
			"Run '.\\d365 connect <url>' to re-authenticate.")
	}
	remaining := time.Until(expiry).Round(time.Minute)
	return pass("token", fmt.Sprintf("Token valid for %s (expires %s)", remaining, sess.TokenExpiry))
}

func checkDaemonStatus() checkResult {
	if !daemon.IsRunning() {
		return warn("daemon", "Daemon is not running",
			"The daemon starts automatically when needed. Run '.\\d365 daemon start' to start manually.")
	}
	pid, err := daemon.ReadPID()
	if err != nil {
		return pass("daemon", "Daemon is running")
	}
	return pass("daemon", fmt.Sprintf("Daemon is running (PID %d)", pid))
}

func checkAuthProfile() checkResult {
	cfg, err := config.Load()
	if err != nil {
		return warn("auth_profile", "Could not load config to check auth profile",
			"Run 'd365 connect <url>' to create a config file.")
	}
	profile := cfg.ActiveProfile(flagProfile)
	method := profile.AuthMethod
	if method == "" {
		method = auth.MethodAzCLI
	}

	switch method {
	case auth.MethodClientCredential:
		if os.Getenv(config.EnvClientSecret) == "" {
			return warn("auth_profile",
				"Profile uses client-credentials auth but D365_CLIENT_SECRET is not set",
				"Set the D365_CLIENT_SECRET environment variable so commands can re-authenticate.\n"+
					"  PowerShell:   $env:D365_CLIENT_SECRET = \"your-secret\"\n"+
					"  Bash / Linux: export D365_CLIENT_SECRET=\"your-secret\"")
		}
		if profile.TenantID == "" {
			return warn("auth_profile",
				"Profile uses client-credentials auth but tenant ID is missing",
				"Reconnect with: d365 connect <url> --auth client-credentials --tenant <id> --client-id <id> --client-secret <secret>")
		}
		if profile.ClientID == "" {
			return warn("auth_profile",
				"Profile uses client-credentials auth but client ID is missing",
				"Reconnect with: d365 connect <url> --auth client-credentials --tenant <id> --client-id <id> --client-secret <secret>")
		}
		return pass("auth_profile", "Client-credentials profile OK (tenant, client ID set, secret in env)")
	case auth.MethodBrowser, auth.MethodDeviceCode:
		if profile.TenantID == "" || profile.ClientID == "" {
			return warn("auth_profile",
				fmt.Sprintf("Profile uses %s auth but tenant or client ID is missing", method),
				fmt.Sprintf("Reconnect with: d365 connect <url> --auth %s --tenant <id> --client-id <id>", method))
		}
		return pass("auth_profile", fmt.Sprintf("%s profile OK (tenant, client ID set)", method))
	default:
		return pass("auth_profile", fmt.Sprintf("Auth method: %s (no extra config needed)", method))
	}
}
