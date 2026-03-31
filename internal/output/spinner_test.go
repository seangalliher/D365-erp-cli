package output

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestSpinner_SuppressedIsNoop(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := NewSpinner(true)
	s.writer = &buf

	s.Start("should not appear")
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	if buf.Len() != 0 {
		t.Errorf("expected no output when suppressed, got %q", buf.String())
	}
}

func TestSpinner_StartsAndStops(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := NewSpinner(false)
	s.writer = &buf

	s.Start("loading")
	time.Sleep(250 * time.Millisecond) // enough for at least 2 frames
	s.Stop()

	out := buf.String()
	if len(out) == 0 {
		t.Error("expected spinner output, got empty string")
	}
	if !containsSubstring(out, "loading") {
		t.Errorf("expected output to contain 'loading', got %q", out)
	}
}

func TestSpinner_UpdateChangesMessage(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := NewSpinner(false)
	s.writer = &buf

	s.Start("first")
	time.Sleep(150 * time.Millisecond)
	s.Update("second")
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !containsSubstring(out, "second") {
		t.Errorf("expected output to contain 'second' after Update, got %q", out)
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := NewSpinner(false)
	s.writer = &buf

	s.Start("test")
	time.Sleep(150 * time.Millisecond)
	s.Stop()
	s.Stop() // should not panic
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	t.Parallel()
	s := NewSpinner(false)
	s.Stop() // should not panic
}

func TestWithSpinner_ReturnsError(t *testing.T) {
	t.Parallel()
	want := errors.New("test error")
	got := WithSpinner(true, "msg", func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("expected error %v, got %v", want, got)
	}
}

func TestWithSpinner_ReturnsNilOnSuccess(t *testing.T) {
	t.Parallel()
	err := WithSpinner(true, "msg", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func containsSubstring(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
