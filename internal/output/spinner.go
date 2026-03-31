package output

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Spinner displays an animated spinner on stderr during long-running operations.
// It is a no-op when suppressed (non-TTY, quiet mode, CI mode).
type Spinner struct {
	mu       sync.Mutex
	message  string
	running  bool
	suppress bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	writer   io.Writer // defaults to os.Stderr; overridable for tests
	lastLen  int       // length of last printed line (for clearing)
}

// frames returns the animation frames appropriate for the current terminal.
func frames() []string {
	if runtime.GOOS == "windows" {
		// Windows Terminal and VS Code terminal support Unicode.
		if os.Getenv("WT_SESSION") != "" || os.Getenv("TERM_PROGRAM") != "" {
			return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		}
		return []string{"|", "/", "-", "\\"}
	}
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}

// NewSpinner creates a new spinner. If suppress is true, all methods are no-ops.
func NewSpinner(suppress bool) *Spinner {
	return &Spinner{suppress: suppress}
}

// Start begins the spinner animation with the given message.
// If already running, it updates the message instead.
func (s *Spinner) Start(message string) {
	if s.suppress {
		return
	}
	s.mu.Lock()
	if s.running {
		s.message = message
		s.mu.Unlock()
		return
	}
	s.message = message
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.animate()
}

// Update changes the displayed message without restarting the spinner.
func (s *Spinner) Update(message string) {
	if s.suppress {
		return
	}
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop halts the spinner and clears the line. It is safe to call multiple times.
func (s *Spinner) Stop() {
	if s.suppress {
		return
	}
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	<-s.doneCh // wait for goroutine to finish

	s.mu.Lock()
	w := s.out()
	lastLen := s.lastLen
	s.lastLen = 0
	s.mu.Unlock()

	// Clear the spinner line.
	if lastLen > 0 {
		fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", lastLen))
	}
}

// animate runs the spinner animation loop in a goroutine.
func (s *Spinner) animate() {
	defer close(s.doneCh)

	f := frames()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			w := s.out()
			s.mu.Unlock()

			line := fmt.Sprintf("\r%s %s", f[idx%len(f)], msg)
			fmt.Fprint(w, line)
			s.mu.Lock()
			s.lastLen = len(line) - 1 // exclude the \r
			s.mu.Unlock()

			idx++
		}
	}
}

// out returns the writer, defaulting to os.Stderr.
func (s *Spinner) out() io.Writer {
	if s.writer != nil {
		return s.writer
	}
	return os.Stderr
}

// WithSpinner runs fn while showing a spinner with the given message.
// The spinner is always stopped before returning, even on error or panic.
func WithSpinner(suppress bool, message string, fn func() error) error {
	s := NewSpinner(suppress)
	s.Start(message)
	defer s.Stop()
	return fn()
}
