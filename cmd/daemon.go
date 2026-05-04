package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/daemon"
	"github.com/seangalliher/d365-erp-cli/internal/formhandler"
)

func init() {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the form session daemon",
		Long: `The daemon runs in the background to manage stateful form sessions.
It is normally auto-started by 'd365 form' commands, but can be
managed explicitly with these subcommands.`,
	}

	daemonCmd.AddCommand(
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonStatusCmd(),
		newDaemonRestartCmd(),
	)

	rootCmd.AddCommand(daemonCmd)
}

func newDaemonStartCmd() *cobra.Command {
	var foreground bool
	var idleTimeout int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the form daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			if !foreground {
				// Check if already running
				if daemon.IsRunning() {
					RenderSuccess(cmd, map[string]string{
						"status": "already_running",
					}, start)
					return nil
				}

				// Start as background process
				if err := daemon.EnsureDaemon(); err != nil {
					formError(cmd, err, start)
					return nil
				}
				RenderSuccess(cmd, map[string]string{
					"status": "started",
				}, start)
				return nil
			}

			// Foreground mode: run the server directly (used by auto-start)
			return runDaemonForeground(idleTimeout)
		},
	}
	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground (used internally)")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 1800, "Idle timeout in seconds")
	_ = cmd.Flags().MarkHidden("foreground")
	return cmd
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the form daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			if err := daemon.StopDaemon(); err != nil {
				formError(cmd, err, start)
				return nil
			}
			RenderSuccess(cmd, map[string]string{"status": "stopped"}, start)
			return nil
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			client, err := daemon.Connect()
			if err != nil {
				RenderSuccess(cmd, map[string]interface{}{
					"running": false,
				}, start)
				return nil
			}
			defer func() { _ = client.Close() }()

			ping, err := client.Ping()
			if err != nil {
				RenderSuccess(cmd, map[string]interface{}{
					"running": false,
				}, start)
				return nil
			}

			pid, _ := daemon.ReadPID()
			RenderSuccess(cmd, map[string]interface{}{
				"running":        true,
				"uptime_seconds": ping.Uptime,
				"form_open":      ping.FormOpen,
				"form_name":      ping.FormName,
				"pid":            pid,
			}, start)
			return nil
		},
	}
}

func newDaemonRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the form daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			// Stop any running daemon
			_ = daemon.StopDaemon()
			time.Sleep(500 * time.Millisecond)

			// Start new one
			if err := daemon.EnsureDaemon(); err != nil {
				formError(cmd, err, start)
				return nil
			}
			RenderSuccess(cmd, map[string]string{"status": "restarted"}, start)
			return nil
		},
	}
}

// runDaemonForeground runs the daemon server in the current process.
func runDaemonForeground(idleTimeoutSec int) error {
	// Write PID file
	pidPath, err := daemon.PIDFilePath()
	if err == nil {
		_ = os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0600)
	}

	// Load session to get environment URL and company.
	sess, err := config.LoadSession()
	if err != nil || !sess.Connected || sess.Environment == "" {
		return fmt.Errorf("no active D365 session — run 'd365 connect <url>' first")
	}

	// Create the browser-based form handler.
	fh, err := formhandler.New(formhandler.Config{
		EnvURL:  sess.Environment,
		Company: sess.Company,
		Visible: true, // Visible for login; goes headless once session is cached.
	})
	if err != nil {
		return fmt.Errorf("cannot create form handler: %w", err)
	}
	defer fh.Close()

	// Eagerly start the browser session so login happens before any
	// commands are sent. This avoids client read deadline timeouts.
	fmt.Fprintln(os.Stderr, "[d365-daemon] starting browser session (complete login if prompted)...")
	if err := fh.WarmUp(context.Background()); err != nil {
		return fmt.Errorf("browser session failed to start: %w", err)
	}
	fmt.Fprintln(os.Stderr, "[d365-daemon] browser session ready")

	cfg := daemon.ServerConfig{
		IdleTimeout: time.Duration(idleTimeoutSec) * time.Second,
	}

	srv := daemon.NewServer(fh, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		srv.Stop()
	}()

	return srv.ListenAndServe(ctx)
}
