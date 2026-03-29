// Package client provides HTTP and OData client wrappers for the D365 CLI.
// It handles authentication, retries with exponential backoff, and structured
// error reporting for all communication with D365 Finance & Operations.
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

// version is the CLI version used in User-Agent, set once via SetVersion.
var (
	version     = "dev"
	versionOnce sync.Once
)

// SetVersion sets the version string used in the User-Agent header.
// It is safe to call from multiple goroutines; only the first call takes effect.
func SetVersion(v string) {
	versionOnce.Do(func() { version = v })
}

// GetVersion returns the current version string.
func GetVersion() string {
	return version
}

// retryConfig holds the retry policy for transient errors.
var retryConfig = struct {
	maxRetries int
	delays     []time.Duration
}{
	maxRetries: 3,
	delays:     []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
}

// transientStatusCodes are HTTP status codes eligible for retry.
var transientStatusCodes = map[int]bool{
	http.StatusTooManyRequests:     true, // 429
	http.StatusInternalServerError: true, // 500
	http.StatusBadGateway:          true, // 502
	http.StatusServiceUnavailable:  true, // 503
	http.StatusGatewayTimeout:      true, // 504
}

// TokenProvider is a function that returns a bearer token for the given context.
type TokenProvider func(ctx context.Context) (string, error)

// Client is an HTTP client wrapper that provides authentication,
// retry logic, and structured error handling for D365 REST endpoints.
type Client struct {
	baseURL       string
	httpClient    *http.Client
	tokenProvider TokenProvider
	timeout       time.Duration
	verbose       bool
}

// NewClient creates a new HTTP Client for communicating with D365.
// The baseURL should be the D365 environment root (e.g., https://env.operations.dynamics.com).
// The tokenProvider is called before each request to obtain a bearer token.
// The timeout controls per-request timeouts.
func NewClient(baseURL string, tokenProvider TokenProvider, timeout time.Duration) *Client {
	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		tokenProvider: tokenProvider,
		timeout:       timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetVerbose enables or disables verbose request/response logging to stderr.
func (c *Client) SetVerbose(v bool) {
	c.verbose = v
}

// Get performs an HTTP GET request to the given path with optional query parameters.
// Extra headers can be passed to override defaults (e.g., Accept for XML endpoints).
func (c *Client) Get(ctx context.Context, path string, queryParams url.Values, extraHeaders ...map[string]string) (*http.Response, error) {
	fullURL := c.buildURL(path, queryParams)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, clierrors.ConnectionError("failed to create GET request", err)
	}
	for _, h := range extraHeaders {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}
	return c.do(req)
}

// Post performs an HTTP POST request with the given body.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	fullURL := c.buildURL(path, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return nil, clierrors.ConnectionError("failed to create POST request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// Patch performs an HTTP PATCH request with the given body.
func (c *Client) Patch(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	fullURL := c.buildURL(path, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fullURL, body)
	if err != nil {
		return nil, clierrors.ConnectionError("failed to create PATCH request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// Delete performs an HTTP DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	fullURL := c.buildURL(path, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		return nil, clierrors.ConnectionError("failed to create DELETE request", err)
	}
	return c.do(req)
}

// buildURL joins the base URL, path, and optional query parameters.
func (c *Client) buildURL(path string, queryParams url.Values) string {
	u := c.baseURL + "/" + strings.TrimLeft(path, "/")
	if len(queryParams) > 0 {
		u += "?" + queryParams.Encode()
	}
	return u
}

// do executes a request with auth injection, standard headers, retries, and logging.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	// Inject auth header.
	token, err := c.tokenProvider(req.Context())
	if err != nil {
		return nil, clierrors.AuthError("failed to obtain access token", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Standard headers.
	req.Header.Set("User-Agent", "d365-cli/"+GetVersion())
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	// Capture body for retries (POST/PATCH need body replay).
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, clierrors.ConnectionError("failed to read request body", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= retryConfig.maxRetries; attempt++ {
		if attempt > 0 {
			// Reset the body for the retry.
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			delay := retryConfig.delays[attempt-1]
			if c.verbose {
				log.Printf("[d365] retry %d/%d after %v for %s %s\n",
					attempt, retryConfig.maxRetries, delay, req.Method, req.URL.String())
			}

			select {
			case <-time.After(delay):
			case <-req.Context().Done():
				return nil, clierrors.TimeoutError(fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			}
		}

		if c.verbose {
			log.Printf("[d365] --> %s %s\n", req.Method, req.URL.String())
		}

		resp, lastErr = c.httpClient.Do(req)

		if lastErr != nil {
			if c.verbose {
				log.Printf("[d365] <-- error: %v\n", lastErr)
			}
			if req.Context().Err() != nil {
				return nil, clierrors.TimeoutError(fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			}
			// Network errors are retryable.
			continue
		}

		if c.verbose {
			log.Printf("[d365] <-- %d %s\n", resp.StatusCode, resp.Status)
		}

		// If the status code is not transient, stop retrying.
		if !transientStatusCodes[resp.StatusCode] {
			break
		}

		// Drain and close the body before retrying.
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("transient HTTP %d", resp.StatusCode)
	}

	if lastErr != nil && resp == nil {
		return nil, clierrors.ConnectionError(
			fmt.Sprintf("request failed after %d retries: %s %s", retryConfig.maxRetries, req.Method, req.URL.Path),
			lastErr,
		)
	}

	// Wrap non-2xx responses as errors while still returning the response
	// so callers can inspect the body.
	if resp.StatusCode >= 400 {
		return resp, classifyHTTPError(resp)
	}

	return resp, nil
}

// classifyHTTPError converts an HTTP error status into a structured CLIError.
func classifyHTTPError(resp *http.Response) *clierrors.CLIError {
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return clierrors.AuthError(
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			nil,
		).WithSuggestion("Token may be expired. Run '.\\d365 connect' to re-authenticate.")
	case resp.StatusCode == http.StatusForbidden:
		return clierrors.Wrap("PERMISSION_DENIED",
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			nil,
		).WithSuggestion("You may lack permissions for this resource. Check your D365 security role.")
	case resp.StatusCode == http.StatusBadRequest:
		return clierrors.Wrap("BAD_REQUEST",
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			nil,
		).WithSuggestion("Check your query syntax. Run '.\\d365 docs odata-filters' for reference.")
	case resp.StatusCode == http.StatusNotFound:
		return clierrors.NotFoundError("resource", resp.Request.URL.Path)
	case resp.StatusCode == http.StatusTooManyRequests:
		return clierrors.Wrap("THROTTLED",
			fmt.Sprintf("HTTP 429: rate limited on %s", resp.Request.URL.Path),
			nil,
		).WithSuggestion("Wait a moment and retry. Consider reducing request frequency.")
	case resp.StatusCode >= 500:
		return clierrors.Wrap("SERVER_ERROR",
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			nil,
		).WithSuggestion("The D365 server returned an error. Try again shortly.")
	default:
		return clierrors.Wrap("HTTP_ERROR",
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			nil,
		)
	}
}
