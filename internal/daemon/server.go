package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Server is the daemon IPC server that handles form session commands.
type Server struct {
	mu       sync.RWMutex
	listener net.Listener
	handler  Handler
	startAt  time.Time
	done     chan struct{}
	idle     *time.Timer
	idleMax  time.Duration
	logger   *log.Logger

	// Session state
	formOpen bool
	formName string
}

// Handler processes daemon commands. Implementations bridge to the
// actual D365 backend (MCP server, direct API, etc.).
type Handler interface {
	// Handle processes a command and returns the response data or an error.
	Handle(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error)
}

// HandlerFunc adapts a plain function to the Handler interface.
type HandlerFunc func(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error)

func (f HandlerFunc) Handle(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
	return f(ctx, command, args)
}

// ServerConfig holds daemon server configuration.
type ServerConfig struct {
	IdleTimeout time.Duration
	LogFile     string
}

// NewServer creates a daemon server with the given handler and config.
func NewServer(handler Handler, cfg ServerConfig) *Server {
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}

	logger := log.New(os.Stderr, "[d365-daemon] ", log.LstdFlags|log.Lmsgprefix)
	if cfg.LogFile != "" {
		if f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
			logger = log.New(f, "[d365-daemon] ", log.LstdFlags|log.Lmsgprefix)
		}
	}

	return &Server{
		handler: handler,
		startAt: time.Now(),
		done:    make(chan struct{}),
		idleMax: cfg.IdleTimeout,
		logger:  logger,
	}
}

// ListenAndServe starts the IPC server on the platform-appropriate transport.
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr, err := SocketPath()
	if err != nil {
		return fmt.Errorf("cannot determine socket path: %w", err)
	}

	var listener net.Listener
	if runtime.GOOS == "windows" {
		// On Windows, use TCP on localhost for simplicity
		listener, err = net.Listen("tcp", "127.0.0.1:51365")
	} else {
		// Clean up stale socket
		_ = os.Remove(addr)
		listener, err = net.Listen("unix", addr)
	}
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", addr, err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	s.logger.Printf("daemon started, listening on %s", addr)

	// Set up idle timer
	s.idle = time.NewTimer(s.idleMax)

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Accept loop
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					// Check if the listener was closed (not a transient error).
					if strings.Contains(err.Error(), "use of closed") {
						return
					}
					s.logger.Printf("accept error: %v", err)
					continue
				}
			}
			s.resetIdle()
			go s.handleConn(ctx, conn)
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		s.logger.Printf("received signal %v, shutting down", sig)
	case <-s.idle.C:
		s.logger.Printf("idle timeout (%v), shutting down", s.idleMax)
	case <-s.done:
		s.logger.Printf("shutdown requested")
	case <-ctx.Done():
		s.logger.Printf("context canceled")
	}

	return s.shutdown()
}

func (s *Server) shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.idle != nil {
		s.idle.Stop()
	}

	// Clean up socket file
	if runtime.GOOS != "windows" {
		addr, _ := SocketPath()
		_ = os.Remove(addr)
	}

	// Clean up PID file
	pidPath, _ := PIDFilePath()
	if pidPath != "" {
		_ = os.Remove(pidPath)
	}

	s.logger.Printf("daemon stopped")
	return nil
}

// Stop triggers a graceful shutdown.
func (s *Server) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *Server) resetIdle() {
	if s.idle != nil && s.idleMax > 0 {
		s.idle.Reset(s.idleMax)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		s.resetIdle()

		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			s.writeError(conn, "", "INVALID_REQUEST", "invalid JSON request")
			continue
		}

		resp := s.processRequest(ctx, &req)

		data, err := json.Marshal(resp)
		if err != nil {
			s.logger.Printf("cannot marshal response: %v", err)
			continue
		}
		if _, err := conn.Write(append(data, '\n')); err != nil {
			s.logger.Printf("write error: %v", err)
			return
		}
	}
}

func (s *Server) processRequest(ctx context.Context, req *Request) *Response {
	switch req.Command {
	case CmdPing:
		return s.handlePing(req)
	case CmdShutdown:
		s.logger.Printf("shutdown command received")
		defer s.Stop()
		return &Response{ID: req.ID, Success: true}
	default:
		return s.handleCommand(ctx, req)
	}
}

func (s *Server) handlePing(req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ping := PingResponse{
		Status:   "ok",
		Uptime:   int64(time.Since(s.startAt).Seconds()),
		FormOpen: s.formOpen,
		FormName: s.formName,
	}
	data, _ := json.Marshal(ping)
	return &Response{ID: req.ID, Success: true, Data: data}
}

func (s *Server) handleCommand(ctx context.Context, req *Request) *Response {
	if s.handler == nil {
		return &Response{
			ID:      req.ID,
			Success: false,
			Error:   &ErrorDetail{Code: "NO_HANDLER", Message: "no command handler configured"},
		}
	}

	result, err := s.handler.Handle(ctx, req.Command, req.Args)
	if err != nil {
		return &Response{
			ID:      req.ID,
			Success: false,
			Error:   &ErrorDetail{Code: "COMMAND_ERROR", Message: err.Error()},
		}
	}

	// Track form state changes
	s.trackFormState(req.Command, result)

	return &Response{ID: req.ID, Success: true, Data: result}
}

func (s *Server) trackFormState(command string, _ json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch command {
	case CmdFormOpen:
		s.formOpen = true
	case CmdFormClose:
		s.formOpen = false
		s.formName = ""
	}
}

func (s *Server) writeError(conn net.Conn, id, code, msg string) {
	resp := &Response{
		ID:      id,
		Success: false,
		Error:   &ErrorDetail{Code: code, Message: msg},
	}
	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

// SetFormState updates the server's tracked form state (used by handlers).
func (s *Server) SetFormState(open bool, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.formOpen = open
	s.formName = name
}
