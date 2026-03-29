package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

// stubTokenProvider returns a fixed token for testing.
func stubTokenProvider(_ context.Context) (string, error) {
	return "test-token-abc", nil
}

// failingTokenProvider always returns an error.
func failingTokenProvider(_ context.Context) (string, error) {
	return "", fmt.Errorf("token refresh failed")
}

func TestClient_RetryOn429ThenSuccess(t *testing.T) {

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"throttled"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"result":"ok"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 10*time.Second)
	// Shorten retry delays for tests.
	origDelays := retryConfig.delays
	retryConfig.delays = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}
	defer func() { retryConfig.delays = origDelays }()

	resp, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("unexpected body: %s", string(body))
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestClient_RetryExhausted(t *testing.T) {

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":"unavailable"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 10*time.Second)
	origDelays := retryConfig.delays
	retryConfig.delays = []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 15 * time.Millisecond}
	defer func() { retryConfig.delays = origDelays }()

	resp, err := c.Get(context.Background(), "/test", nil)
	// After all retries, it should return the last 503 response with an error.
	if err == nil {
		t.Fatal("expected error after retry exhaustion")
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Should have tried 1 initial + 3 retries = 4 total.
	if got := atomic.LoadInt32(&attempts); got != 4 {
		t.Errorf("expected 4 attempts, got %d", got)
	}
}

func TestClient_AuthHeaderInjected(t *testing.T) {

	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/data/Customers", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	expected := "Bearer test-token-abc"
	if capturedAuth != expected {
		t.Errorf("expected Authorization %q, got %q", expected, capturedAuth)
	}
}

func TestClient_UserAgentHeader(t *testing.T) {

	var capturedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if !strings.HasPrefix(capturedUA, "d365-cli/") {
		t.Errorf("expected User-Agent starting with 'd365-cli/', got %q", capturedUA)
	}
}

func TestClient_AcceptHeaderDefault(t *testing.T) {

	var capturedAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if capturedAccept != "application/json" {
		t.Errorf("expected Accept 'application/json', got %q", capturedAccept)
	}
}

func TestClient_TokenProviderError(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not have been called")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, failingTokenProvider, 5*time.Second)
	_, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error from failing token provider")
	}
	if !strings.Contains(err.Error(), "AUTH_ERROR") {
		t.Errorf("expected AUTH_ERROR, got: %v", err)
	}
}

func TestClient_Timeout(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 100*time.Millisecond)
	origDelays := retryConfig.delays
	retryConfig.delays = []time.Duration{5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond}
	defer func() { retryConfig.delays = origDelays }()

	_, err := c.Get(context.Background(), "/slow", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "CONNECTION_ERROR") && !strings.Contains(errStr, "TIMEOUT") {
		t.Errorf("expected CONNECTION_ERROR or TIMEOUT, got: %v", err)
	}
}

func TestClient_ErrorWrapping_401(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if resp != nil {
		resp.Body.Close()
	}
	if !strings.Contains(err.Error(), "AUTH_ERROR") {
		t.Errorf("expected AUTH_ERROR, got: %v", err)
	}
}

func TestClient_ErrorWrapping_404(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/data/NonExistent", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if resp != nil {
		resp.Body.Close()
	}
	if !strings.Contains(err.Error(), "NOT_FOUND") {
		t.Errorf("expected NOT_FOUND, got: %v", err)
	}
}

func TestClient_PostBodySent(t *testing.T) {

	var capturedBody string
	var capturedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	body := strings.NewReader(`{"name":"Test"}`)
	resp, err := c.Post(context.Background(), "/data/Customers", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if capturedContentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", capturedContentType)
	}
	if capturedBody != `{"name":"Test"}` {
		t.Errorf("unexpected body: %s", capturedBody)
	}
}

func TestClient_ContextCancellation(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := NewClient(srv.URL, stubTokenProvider, 10*time.Second)
	origDelays := retryConfig.delays
	retryConfig.delays = []time.Duration{5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond}
	defer func() { retryConfig.delays = origDelays }()

	_, err := c.Get(ctx, "/slow", nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestClient_DeleteMethod(t *testing.T) {

	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Delete(context.Background(), "/data/Customers('US-001')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if capturedMethod != http.MethodDelete {
		t.Errorf("expected DELETE method, got %s", capturedMethod)
	}
}

func TestClient_ErrorWrapping_403_PermissionDenied(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"forbidden"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if resp != nil {
		resp.Body.Close()
	}
	if !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Errorf("expected PERMISSION_DENIED, got: %v", err)
	}
}

func TestClient_ErrorWrapping_400_BadRequest(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad request"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	resp, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if resp != nil {
		resp.Body.Close()
	}
	if !strings.Contains(err.Error(), "BAD_REQUEST") {
		t.Errorf("expected BAD_REQUEST, got: %v", err)
	}
}

func TestClient_ErrorSuggestions(t *testing.T) {

	tests := []struct {
		name       string
		statusCode int
		wantHint   string
	}{
		{"401 suggests reconnect", http.StatusUnauthorized, "re-authenticate"},
		{"403 suggests permissions", http.StatusForbidden, "security role"},
		{"400 suggests docs", http.StatusBadRequest, "odata-filters"},
		{"404 suggests find-type", http.StatusNotFound, "find-type"},
		{"429 suggests wait", http.StatusTooManyRequests, "Wait"},
		{"500 suggests retry", http.StatusInternalServerError, "Try again"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, `{"error":"test"}`)
			}))
			defer srv.Close()

			c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
			origDelays := retryConfig.delays
			retryConfig.delays = []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond}
			defer func() { retryConfig.delays = origDelays }()

			resp, err := c.Get(context.Background(), "/test", nil)
			if resp != nil {
				resp.Body.Close()
			}
			if err == nil {
				t.Fatal("expected error")
			}

			cliErr, ok := err.(*clierrors.CLIError)
			if !ok {
				t.Fatalf("expected *CLIError, got %T", err)
			}
			if !strings.Contains(cliErr.Suggestion, tt.wantHint) {
				t.Errorf("suggestion %q should contain %q", cliErr.Suggestion, tt.wantHint)
			}
		})
	}
}

func TestClient_PatchMethod(t *testing.T) {

	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, stubTokenProvider, 5*time.Second)
	body := strings.NewReader(`{"Name":"Updated"}`)
	resp, err := c.Patch(context.Background(), "/data/Customers('US-001')", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH method, got %s", capturedMethod)
	}
}
