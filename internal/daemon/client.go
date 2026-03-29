package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/seangalliher/d365-erp-cli/internal/config"
	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

// Client communicates with the daemon over IPC (named pipe or Unix socket).
type Client struct {
	mu   sync.Mutex
	conn net.Conn
}

// SocketPath returns the IPC socket/pipe path for the current user.
func SocketPath() (string, error) {
	dir, err := config.ConfigDirPath()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return `\\.\pipe\d365cli-daemon`, nil
	}
	return filepath.Join(dir, "daemon.sock"), nil
}

// PIDFilePath returns the path to the daemon PID file.
func PIDFilePath() (string, error) {
	dir, err := config.ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// Connect establishes a connection to the running daemon.
func Connect() (*Client, error) {
	addr, err := SocketPath()
	if err != nil {
		return nil, clierrors.DaemonError("cannot determine socket path", err)
	}

	network := "unix"
	if runtime.GOOS == "windows" {
		network = "pipe"
	}

	conn, err := dialIPC(network, addr)
	if err != nil {
		return nil, clierrors.DaemonError("cannot connect to daemon", err).
			WithSuggestion("Is the daemon running? Try 'd365 form open' to auto-start it.")
	}
	return &Client{conn: conn}, nil
}

// dialIPC wraps the platform-specific dial. On Windows we use net.Dial("pipe",...),
// on Unix we use net.Dial("unix", ...).
func dialIPC(network, addr string) (net.Conn, error) {
	// net.Dial supports "unix" natively. On Windows, named pipes require
	// a specialized library for full support. For now we use a simple TCP
	// fallback on Windows to keep dependencies minimal.
	if network == "pipe" {
		// Windows named pipe: try to connect via DialTimeout since pipe
		// names are not directly supported by Go's net.Dial.
		// We use a local TCP port as the transport for simplicity.
		return net.DialTimeout("tcp", "127.0.0.1:"+windowsPort(addr), 5*time.Second)
	}
	return net.DialTimeout("unix", addr, 5*time.Second)
}

// windowsPort extracts or derives a port from the pipe name for TCP fallback.
func windowsPort(pipeName string) string {
	// Use a consistent port derived from pipe name hash.
	// Default daemon port: 51365 (d365).
	_ = pipeName
	return "51365"
}

// Close disconnects from the daemon.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Send sends a request and waits for a response. Thread-safe.
func (c *Client) Send(req *Request) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, clierrors.DaemonError("not connected to daemon", nil)
	}

	// Write newline-delimited JSON
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal request: %w", err)
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("cannot set write deadline: %w", err)
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, clierrors.DaemonError("failed to send command to daemon", err)
	}

	// Read response
	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return nil, fmt.Errorf("cannot set read deadline: %w", err)
	}
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		if scanner.Err() != nil {
			return nil, clierrors.DaemonError("failed to read daemon response", scanner.Err())
		}
		return nil, clierrors.DaemonError("daemon closed connection unexpectedly", nil)
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, clierrors.DaemonError("invalid daemon response", err)
	}
	return &resp, nil
}

// Ping checks if the daemon is alive.
func (c *Client) Ping() (*PingResponse, error) {
	req := &Request{ID: newID(), Command: CmdPing}
	resp, err := c.Send(req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, daemonRespError(resp)
	}
	var ping PingResponse
	if err := json.Unmarshal(resp.Data, &ping); err != nil {
		return nil, fmt.Errorf("cannot parse ping response: %w", err)
	}
	return &ping, nil
}

// SendCommand sends an arbitrary command with typed args to the daemon.
func (c *Client) SendCommand(command string, args interface{}) (*Response, error) {
	var rawArgs json.RawMessage
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal args: %w", err)
		}
		rawArgs = b
	}

	req := &Request{
		ID:      newID(),
		Command: command,
		Args:    rawArgs,
	}
	resp, err := c.Send(req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, daemonRespError(resp)
	}
	return resp, nil
}

// EnsureDaemon starts the daemon if it is not already running.
func EnsureDaemon() error {
	// Try to connect first
	client, err := Connect()
	if err == nil {
		_, pingErr := client.Ping()
		_ = client.Close()
		if pingErr == nil {
			return nil // Already running
		}
	}

	// Start daemon
	return startDaemon()
}

func startDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return clierrors.DaemonError("cannot determine executable path", err)
	}

	// The daemon runs as a subprocess in a detached process group.
	cmd := exec.Command(exe, "daemon", "start", "--foreground")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return clierrors.DaemonError("failed to start daemon", err)
	}

	// Save PID
	pidPath, err := PIDFilePath()
	if err == nil {
		_ = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0600)
	}

	// Release the process so it runs independently
	_ = cmd.Process.Release()

	// Wait for daemon to become ready
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		c, err := Connect()
		if err != nil {
			continue
		}
		_, err = c.Ping()
		_ = c.Close()
		if err == nil {
			return nil
		}
	}
	return clierrors.DaemonError("daemon did not start within 6 seconds", nil)
}

// StopDaemon sends a shutdown command to the running daemon.
func StopDaemon() error {
	client, err := Connect()
	if err != nil {
		return nil // Already stopped
	}
	defer func() { _ = client.Close() }()

	_, _ = client.SendCommand(CmdShutdown, nil)

	// Clean up PID file
	pidPath, _ := PIDFilePath()
	if pidPath != "" {
		_ = os.Remove(pidPath)
	}
	return nil
}

// IsRunning checks whether the daemon is alive.
func IsRunning() bool {
	client, err := Connect()
	if err != nil {
		return false
	}
	defer func() { _ = client.Close() }()
	_, err = client.Ping()
	return err == nil
}

func daemonRespError(resp *Response) error {
	if resp.Error != nil {
		return clierrors.New(resp.Error.Code, resp.Error.Message)
	}
	return clierrors.DaemonError("unknown daemon error", nil)
}

var idCounter int64
var idMu sync.Mutex

func newID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return fmt.Sprintf("req-%d-%d", time.Now().UnixMilli(), idCounter)
}

// ReadPID reads the daemon PID from the PID file.
func ReadPID() (int, error) {
	pidPath, err := PIDFilePath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, nil
}
