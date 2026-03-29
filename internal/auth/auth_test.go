package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seangalliher/d365-erp-cli/internal/errors"
)

// ---------------------------------------------------------------------------
// fakeProvider is a test double for Provider.
// ---------------------------------------------------------------------------

type fakeProvider struct {
	method string
	token  *TokenResult
	err    error
}

func (f *fakeProvider) GetToken(_ context.Context, _ []string) (*TokenResult, error) {
	return f.token, f.err
}

func (f *fakeProvider) Method() string { return f.method }

// fakeFactory builds a CredentialFactory that returns fakeProviders.
func fakeFactory(token *TokenResult, err error) CredentialFactory {
	return func(method string, _ ProviderConfig) (Provider, error) {
		return &fakeProvider{method: method, token: token, err: err}, nil
	}
}

// fakeFactoryError returns a factory whose construction itself fails.
func fakeFactoryError(err error) CredentialFactory {
	return func(_ string, _ ProviderConfig) (Provider, error) {
		return nil, err
	}
}

// ---------------------------------------------------------------------------
// NewAuthenticator tests
// ---------------------------------------------------------------------------

func TestNewAuthenticator_AllValidMethods(t *testing.T) {
	t.Parallel()

	methods := []string{
		MethodBrowser,
		MethodDeviceCode,
		MethodClientCredential,
		MethodAzCLI,
		MethodManagedIdentity,
	}

	tok := &TokenResult{
		AccessToken: "fake-token",
		ExpiresOn:   time.Now().Add(time.Hour),
	}

	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			t.Parallel()

			auth, err := NewAuthenticatorWithFactory(m, ProviderConfig{}, fakeFactory(tok, nil))
			if err != nil {
				t.Fatalf("NewAuthenticatorWithFactory(%q) unexpected error: %v", m, err)
			}
			if auth.Method() != m {
				t.Errorf("Method() = %q, want %q", auth.Method(), m)
			}
		})
	}
}

func TestNewAuthenticator_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tok := &TokenResult{AccessToken: "tok", ExpiresOn: time.Now().Add(time.Hour)}

	variants := []string{"Browser", "BROWSER", " browser ", "AZ-CLI", " Device-Code "}
	for _, v := range variants {
		t.Run(v, func(t *testing.T) {
			t.Parallel()

			auth, err := NewAuthenticatorWithFactory(v, ProviderConfig{}, fakeFactory(tok, nil))
			if err != nil {
				t.Fatalf("NewAuthenticatorWithFactory(%q) unexpected error: %v", v, err)
			}
			if auth.provider == nil {
				t.Fatal("provider should not be nil")
			}
		})
	}
}

func TestNewAuthenticator_InvalidMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
	}{
		{"empty", ""},
		{"typo", "browzer"},
		{"unknown", "saml"},
		{"partial", "client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewAuthenticatorWithFactory(tt.method, ProviderConfig{}, fakeFactory(nil, nil))
			if err == nil {
				t.Fatalf("expected error for method %q, got nil", tt.method)
			}

			cliErr, ok := err.(*errors.CLIError)
			if !ok {
				t.Fatalf("expected *errors.CLIError, got %T", err)
			}
			if cliErr.Code != "VALIDATION_ERROR" {
				t.Errorf("expected VALIDATION_ERROR code, got %q", cliErr.Code)
			}
			if !strings.Contains(cliErr.Message, "unsupported auth method") {
				t.Errorf("error message should mention 'unsupported auth method', got %q", cliErr.Message)
			}
		})
	}
}

func TestNewAuthenticator_FactoryError(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("sdk init failed")
	_, err := NewAuthenticatorWithFactory(MethodAzCLI, ProviderConfig{}, fakeFactoryError(cause))
	if err == nil {
		t.Fatal("expected error when factory fails")
	}
	if !strings.Contains(err.Error(), "sdk init failed") {
		t.Errorf("error should contain cause, got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// DefaultScope tests
// ---------------------------------------------------------------------------

func TestDefaultScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "plain url",
			url:  "https://contoso.operations.dynamics.com",
			want: "https://contoso.operations.dynamics.com/.default",
		},
		{
			name: "trailing slash",
			url:  "https://contoso.operations.dynamics.com/",
			want: "https://contoso.operations.dynamics.com/.default",
		},
		{
			name: "multiple trailing slashes",
			url:  "https://contoso.operations.dynamics.com///",
			want: "https://contoso.operations.dynamics.com/.default",
		},
		{
			name: "sandbox url",
			url:  "https://contoso-sandbox.sandbox.operations.dynamics.com",
			want: "https://contoso-sandbox.sandbox.operations.dynamics.com/.default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DefaultScope(tt.url)
			if got != tt.want {
				t.Errorf("DefaultScope(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TokenResult tests
// ---------------------------------------------------------------------------

func TestTokenResult_Validate_Valid(t *testing.T) {
	t.Parallel()

	tok := &TokenResult{
		AccessToken: "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUz...",
		ExpiresOn:   time.Now().Add(time.Hour),
	}
	if err := tok.Validate(); err != nil {
		t.Errorf("Validate() on valid token: unexpected error: %v", err)
	}
}

func TestTokenResult_Validate_Nil(t *testing.T) {
	t.Parallel()

	var tok *TokenResult
	err := tok.Validate()
	if err == nil {
		t.Fatal("Validate() on nil token should return error")
	}
	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.Code != "AUTH_ERROR" {
		t.Errorf("expected AUTH_ERROR, got %q", cliErr.Code)
	}
}

func TestTokenResult_Validate_EmptyAccessToken(t *testing.T) {
	t.Parallel()

	tok := &TokenResult{
		AccessToken: "",
		ExpiresOn:   time.Now().Add(time.Hour),
	}
	err := tok.Validate()
	if err == nil {
		t.Fatal("Validate() with empty token should return error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got %q", err.Error())
	}
}

func TestTokenResult_Validate_ZeroExpiry(t *testing.T) {
	t.Parallel()

	tok := &TokenResult{
		AccessToken: "some-token",
		ExpiresOn:   time.Time{},
	}
	err := tok.Validate()
	if err == nil {
		t.Fatal("Validate() with zero expiry should return error")
	}
	if !strings.Contains(err.Error(), "expiration") {
		t.Errorf("error should mention 'expiration', got %q", err.Error())
	}
}

func TestTokenResult_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   *TokenResult
		grace   time.Duration
		expired bool
	}{
		{
			name:    "nil token",
			token:   nil,
			grace:   0,
			expired: true,
		},
		{
			name:    "empty access token",
			token:   &TokenResult{AccessToken: "", ExpiresOn: time.Now().Add(time.Hour)},
			grace:   0,
			expired: true,
		},
		{
			name:    "future expiry no grace",
			token:   &TokenResult{AccessToken: "tok", ExpiresOn: time.Now().Add(time.Hour)},
			grace:   0,
			expired: false,
		},
		{
			name:    "future expiry within grace",
			token:   &TokenResult{AccessToken: "tok", ExpiresOn: time.Now().Add(2 * time.Minute)},
			grace:   5 * time.Minute,
			expired: true,
		},
		{
			name:    "past expiry",
			token:   &TokenResult{AccessToken: "tok", ExpiresOn: time.Now().Add(-time.Hour)},
			grace:   0,
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.token.IsExpired(tt.grace)
			if got != tt.expired {
				t.Errorf("IsExpired(%v) = %v, want %v", tt.grace, got, tt.expired)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetToken / GetD365Token tests
// ---------------------------------------------------------------------------

func TestAuthenticator_GetToken_Success(t *testing.T) {
	t.Parallel()

	expected := &TokenResult{
		AccessToken: "access-123",
		ExpiresOn:   time.Now().Add(time.Hour),
	}

	auth, err := NewAuthenticatorWithFactory(MethodAzCLI, ProviderConfig{}, fakeFactory(expected, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tok, err := auth.GetToken(context.Background(), []string{"https://contoso.operations.dynamics.com/.default"})
	if err != nil {
		t.Fatalf("GetToken() unexpected error: %v", err)
	}
	if tok.AccessToken != expected.AccessToken {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, expected.AccessToken)
	}
}

func TestAuthenticator_GetToken_ProviderError(t *testing.T) {
	t.Parallel()

	provErr := fmt.Errorf("credential refresh failed")
	auth, err := NewAuthenticatorWithFactory(MethodBrowser, ProviderConfig{}, fakeFactory(nil, provErr))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = auth.GetToken(context.Background(), []string{"scope"})
	if err == nil {
		t.Fatal("expected error from GetToken when provider fails")
	}

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.Code != "AUTH_ERROR" {
		t.Errorf("expected AUTH_ERROR, got %q", cliErr.Code)
	}
	if cliErr.Suggestion == "" {
		t.Error("CLIError should have a suggestion")
	}
}

func TestAuthenticator_GetD365Token_Success(t *testing.T) {
	t.Parallel()

	expected := &TokenResult{
		AccessToken: "d365-token",
		ExpiresOn:   time.Now().Add(time.Hour),
	}

	cfg := ProviderConfig{
		Environment: "https://contoso.operations.dynamics.com",
	}

	auth, err := NewAuthenticatorWithFactory(MethodAzCLI, cfg, fakeFactory(expected, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tok, err := auth.GetD365Token(context.Background())
	if err != nil {
		t.Fatalf("GetD365Token() unexpected error: %v", err)
	}
	if tok.AccessToken != expected.AccessToken {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, expected.AccessToken)
	}
}

func TestAuthenticator_GetD365Token_NoEnvironment(t *testing.T) {
	t.Parallel()

	auth, err := NewAuthenticatorWithFactory(MethodAzCLI, ProviderConfig{}, fakeFactory(nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = auth.GetD365Token(context.Background())
	if err == nil {
		t.Fatal("expected error when environment is empty")
	}
	if !strings.Contains(err.Error(), "environment URL") {
		t.Errorf("error should mention environment URL, got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Provider() accessor test
// ---------------------------------------------------------------------------

func TestAuthenticator_Provider(t *testing.T) {
	t.Parallel()

	tok := &TokenResult{AccessToken: "tok", ExpiresOn: time.Now().Add(time.Hour)}
	auth, err := NewAuthenticatorWithFactory(MethodManagedIdentity, ProviderConfig{}, fakeFactory(tok, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := auth.Provider()
	if p == nil {
		t.Fatal("Provider() should not return nil")
	}
	if p.Method() != MethodManagedIdentity {
		t.Errorf("Provider().Method() = %q, want %q", p.Method(), MethodManagedIdentity)
	}
}

// ---------------------------------------------------------------------------
// Method constants test
// ---------------------------------------------------------------------------

func TestMethodConstants(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		"browser":            MethodBrowser,
		"device-code":        MethodDeviceCode,
		"client-credentials": MethodClientCredential,
		"az-cli":             MethodAzCLI,
		"managed-identity":   MethodManagedIdentity,
	}

	for want, got := range expected {
		if got != want {
			t.Errorf("constant mismatch: got %q, want %q", got, want)
		}
	}

	if len(validMethods) != len(expected) {
		t.Errorf("validMethods has %d entries, want %d", len(validMethods), len(expected))
	}
}

// ---------------------------------------------------------------------------
// Default factory validation tests (without Azure SDK)
// ---------------------------------------------------------------------------

func TestDefaultFactory_BrowserRequiresTenantAndClient(t *testing.T) {
	t.Parallel()

	// Missing tenant
	_, err := newAzureProvider(MethodBrowser, ProviderConfig{ClientID: "app"})
	if err == nil {
		t.Fatal("expected error when tenant-id is missing for browser auth")
	}

	// Missing client
	_, err = newAzureProvider(MethodBrowser, ProviderConfig{TenantID: "tenant"})
	if err == nil {
		t.Fatal("expected error when client-id is missing for browser auth")
	}
}

func TestDefaultFactory_DeviceCodeRequiresTenantAndClient(t *testing.T) {
	t.Parallel()

	_, err := newAzureProvider(MethodDeviceCode, ProviderConfig{ClientID: "app"})
	if err == nil {
		t.Fatal("expected error when tenant-id is missing for device-code auth")
	}

	_, err = newAzureProvider(MethodDeviceCode, ProviderConfig{TenantID: "tenant"})
	if err == nil {
		t.Fatal("expected error when client-id is missing for device-code auth")
	}
}

func TestDefaultFactory_ClientCredentialsRequiresAllThree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  ProviderConfig
	}{
		{"missing tenant", ProviderConfig{ClientID: "app", ClientSecret: "secret"}},
		{"missing client", ProviderConfig{TenantID: "t", ClientSecret: "secret"}},
		{"missing secret", ProviderConfig{TenantID: "t", ClientID: "app"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newAzureProvider(MethodClientCredential, tt.cfg)
			if err == nil {
				t.Fatalf("expected validation error for %s", tt.name)
			}
			cliErr, ok := err.(*errors.CLIError)
			if !ok {
				t.Fatalf("expected *errors.CLIError, got %T", err)
			}
			if cliErr.Code != "VALIDATION_ERROR" {
				t.Errorf("expected VALIDATION_ERROR, got %q", cliErr.Code)
			}
		})
	}
}

func TestDefaultFactory_UnsupportedMethod(t *testing.T) {
	t.Parallel()

	_, err := newAzureProvider("kerberos", ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for unsupported method")
	}
}

func TestMethodNames_Sorted(t *testing.T) {
	t.Parallel()

	names := methodNames()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("methodNames() not sorted: %q comes after %q", names[i], names[i-1])
		}
	}
}
