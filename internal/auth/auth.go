// Package auth provides authentication for D365 Finance & Operations environments.
// It supports multiple auth methods (browser, device-code, client-credentials,
// az-cli, managed-identity) through a pluggable Provider interface that wraps
// the Azure Identity SDK for testability.
package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/seangalliher/d365-erp-cli/internal/errors"
)

// D365 resource scope suffix used when constructing the OAuth scope.
const d365ScopeSuffix = "/.default"

// Method constants for supported authentication methods.
const (
	MethodBrowser          = "browser"
	MethodDeviceCode       = "device-code"
	MethodClientCredential = "client-credentials"
	MethodAzCLI            = "az-cli"
	MethodManagedIdentity  = "managed-identity"
)

// validMethods is the set of recognized auth method strings.
var validMethods = map[string]bool{
	MethodBrowser:          true,
	MethodDeviceCode:       true,
	MethodClientCredential: true,
	MethodAzCLI:            true,
	MethodManagedIdentity:  true,
}

// TokenResult holds an OAuth access token and its expiration time.
type TokenResult struct {
	// AccessToken is the bearer token string.
	AccessToken string
	// ExpiresOn is the UTC time at which the token expires.
	ExpiresOn time.Time
}

// IsExpired reports whether the token has expired or will expire within the
// given grace period.
func (t *TokenResult) IsExpired(grace time.Duration) bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	return time.Now().UTC().Add(grace).After(t.ExpiresOn)
}

// Validate checks that the token result contains usable data.
func (t *TokenResult) Validate() error {
	if t == nil {
		return errors.AuthError("token result is nil", nil)
	}
	if t.AccessToken == "" {
		return errors.AuthError("access token is empty", nil)
	}
	if t.ExpiresOn.IsZero() {
		return errors.AuthError("token expiration is not set", nil)
	}
	return nil
}

// Provider is the interface every authentication method must implement.
// Implementations wrap a concrete Azure Identity credential so that callers
// and tests are decoupled from the SDK.
type Provider interface {
	// GetToken obtains an access token for the given scopes.
	GetToken(ctx context.Context, scopes []string) (*TokenResult, error)
	// Method returns the name of the auth method (e.g. "browser").
	Method() string
}

// ProviderConfig carries all parameters that any Provider might need.
// Fields that are irrelevant for a given method are simply ignored.
type ProviderConfig struct {
	// TenantID is the Azure AD tenant (required for browser, device-code,
	// client-credentials).
	TenantID string
	// ClientID is the application (client) ID.
	ClientID string
	// ClientSecret is required only for client-credentials.
	ClientSecret string
	// Environment is the D365 environment URL (e.g.
	// "https://contoso.operations.dynamics.com"). Used to derive the
	// resource scope.
	Environment string
	// RedirectURL overrides the default redirect for interactive flows.
	RedirectURL string
	// DeviceCodeHandler is an optional callback invoked with the user code
	// during the device-code flow. If nil, the default handler prints to
	// stderr.
	DeviceCodeHandler func(ctx context.Context, message string)
}

// DefaultScope returns the OAuth scope string for the given D365 environment
// URL. The scope is the environment base URL with "/.default" appended.
func DefaultScope(environmentURL string) string {
	base := strings.TrimRight(environmentURL, "/")
	return base + d365ScopeSuffix
}

// CredentialFactory is a function that creates a Provider for a given method
// and configuration. The default factory (newAzureProvider) constructs real
// Azure Identity credentials; tests can substitute a fake.
type CredentialFactory func(method string, cfg ProviderConfig) (Provider, error)

// Authenticator manages authentication to a D365 F&O environment. It holds
// the selected Provider and offers convenience helpers for obtaining tokens.
type Authenticator struct {
	provider Provider
	config   ProviderConfig
}

// NewAuthenticator creates an Authenticator for the given method string using
// the default Azure Identity credential factory.
func NewAuthenticator(method string, cfg ProviderConfig) (*Authenticator, error) {
	return NewAuthenticatorWithFactory(method, cfg, nil)
}

// NewAuthenticatorWithFactory creates an Authenticator using the supplied
// CredentialFactory. If factory is nil the default Azure Identity factory is
// used.
func NewAuthenticatorWithFactory(method string, cfg ProviderConfig, factory CredentialFactory) (*Authenticator, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	if !validMethods[method] {
		return nil, errors.ValidationError(
			fmt.Sprintf("unsupported auth method %q (valid: %s)",
				method, strings.Join(methodNames(), ", ")),
		)
	}

	if factory == nil {
		factory = newAzureProvider
	}

	p, err := factory(method, cfg)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		provider: p,
		config:   cfg,
	}, nil
}

// GetToken obtains a token for the specified scopes.
func (a *Authenticator) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	if a.provider == nil {
		return nil, errors.AuthError("no auth provider configured", nil)
	}
	tok, err := a.provider.GetToken(ctx, scopes)
	if err != nil {
		return nil, errors.AuthError("failed to acquire token", err).
			WithSuggestion("Run 'd365 connect <url>' to authenticate. Check your credentials or try a different auth method with --auth.")
	}
	return tok, nil
}

// GetD365Token is a convenience method that requests a token scoped to the
// configured D365 environment.
func (a *Authenticator) GetD365Token(ctx context.Context) (*TokenResult, error) {
	if a.config.Environment == "" {
		return nil, errors.AuthError("environment URL is not configured", nil).
			WithSuggestion("Set the environment URL via 'd365 connect <url>' or D365_URL environment variable.")
	}
	scope := DefaultScope(a.config.Environment)
	return a.GetToken(ctx, []string{scope})
}

// Provider returns the underlying Provider.
func (a *Authenticator) Provider() Provider {
	return a.provider
}

// Method returns the name of the active auth method.
func (a *Authenticator) Method() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Method()
}

// ---------------------------------------------------------------------------
// Concrete providers -- each wraps an Azure Identity credential.
// They are constructed by newAzureProvider (the default CredentialFactory).
// ---------------------------------------------------------------------------

// BrowserProvider implements Provider using
// azidentity.InteractiveBrowserCredential.
type BrowserProvider struct {
	credential interface {
		GetTokenFunc(ctx context.Context, scopes []string) (*TokenResult, error)
	}
}

func (p *BrowserProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	return p.credential.GetTokenFunc(ctx, scopes)
}
func (p *BrowserProvider) Method() string { return MethodBrowser }

// DeviceCodeProvider implements Provider using
// azidentity.DeviceCodeCredential.
type DeviceCodeProvider struct {
	credential interface {
		GetTokenFunc(ctx context.Context, scopes []string) (*TokenResult, error)
	}
}

func (p *DeviceCodeProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	return p.credential.GetTokenFunc(ctx, scopes)
}
func (p *DeviceCodeProvider) Method() string { return MethodDeviceCode }

// ClientCredentialProvider implements Provider using
// azidentity.ClientSecretCredential.
type ClientCredentialProvider struct {
	credential interface {
		GetTokenFunc(ctx context.Context, scopes []string) (*TokenResult, error)
	}
}

func (p *ClientCredentialProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	return p.credential.GetTokenFunc(ctx, scopes)
}
func (p *ClientCredentialProvider) Method() string { return MethodClientCredential }

// AzCLIProvider implements Provider using azidentity.AzureCLICredential.
type AzCLIProvider struct {
	credential interface {
		GetTokenFunc(ctx context.Context, scopes []string) (*TokenResult, error)
	}
}

func (p *AzCLIProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	return p.credential.GetTokenFunc(ctx, scopes)
}
func (p *AzCLIProvider) Method() string { return MethodAzCLI }

// ManagedIdentityProvider implements Provider using
// azidentity.ManagedIdentityCredential.
type ManagedIdentityProvider struct {
	credential interface {
		GetTokenFunc(ctx context.Context, scopes []string) (*TokenResult, error)
	}
	clientID string // optional user-assigned identity
}

func (p *ManagedIdentityProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	return p.credential.GetTokenFunc(ctx, scopes)
}
func (p *ManagedIdentityProvider) Method() string { return MethodManagedIdentity }

// ---------------------------------------------------------------------------
// Default credential factory
// ---------------------------------------------------------------------------

// newAzureProvider constructs the real Azure Identity credential for the
// requested method. This is the production CredentialFactory. It validates
// that the required configuration fields are present before attempting to
// create the credential.
//
// NOTE: The actual azidentity calls are behind a thin adapter
// (azureCredentialAdapter) defined in azure_credential.go so that this file
// compiles and tests run without the Azure SDK. If azure_credential.go is not
// present the linker will fail -- a build-tag-gated stub is provided for
// unit-test builds.
func newAzureProvider(method string, cfg ProviderConfig) (Provider, error) {
	switch method {
	case MethodBrowser:
		if cfg.TenantID == "" {
			return nil, errors.ValidationError("--tenant is required for browser auth").
				WithSuggestion("Set --tenant or D365_TENANT_ID.")
		}
		if cfg.ClientID == "" {
			return nil, errors.ValidationError("--client-id is required for browser auth").
				WithSuggestion("Set --client-id or D365_CLIENT_ID.")
		}
		cred, err := newBrowserCredential(cfg)
		if err != nil {
			return nil, errors.AuthError("failed to create browser credential", err)
		}
		return cred, nil

	case MethodDeviceCode:
		if cfg.TenantID == "" {
			return nil, errors.ValidationError("--tenant is required for device-code auth").
				WithSuggestion("Set --tenant or D365_TENANT_ID.")
		}
		if cfg.ClientID == "" {
			return nil, errors.ValidationError("--client-id is required for device-code auth").
				WithSuggestion("Set --client-id or D365_CLIENT_ID.")
		}
		cred, err := newDeviceCodeCredential(cfg)
		if err != nil {
			return nil, errors.AuthError("failed to create device-code credential", err)
		}
		return cred, nil

	case MethodClientCredential:
		if cfg.TenantID == "" {
			return nil, errors.ValidationError("--tenant is required for client-credentials auth").
				WithSuggestion("Set --tenant or D365_TENANT_ID.")
		}
		if cfg.ClientID == "" {
			return nil, errors.ValidationError("--client-id is required for client-credentials auth").
				WithSuggestion("Set --client-id or D365_CLIENT_ID.")
		}
		if cfg.ClientSecret == "" {
			return nil, errors.ValidationError("--client-secret is required for client-credentials auth").
				WithSuggestion("Set --client-secret or D365_CLIENT_SECRET.")
		}
		cred, err := newClientSecretCredential(cfg)
		if err != nil {
			return nil, errors.AuthError("failed to create client-secret credential", err)
		}
		return cred, nil

	case MethodAzCLI:
		cred, err := newAzCLICredential(cfg)
		if err != nil {
			return nil, errors.AuthError("failed to create az-cli credential", err)
		}
		return cred, nil

	case MethodManagedIdentity:
		cred, err := newManagedIdentityCredential(cfg)
		if err != nil {
			return nil, errors.AuthError("failed to create managed-identity credential", err)
		}
		return cred, nil

	default:
		return nil, errors.ValidationError(
			fmt.Sprintf("unsupported auth method: %s", method),
		)
	}
}

// ---------------------------------------------------------------------------
// Credential constructor stubs -- overridden by azure_credential.go in
// production builds and by test fakes in tests.
// ---------------------------------------------------------------------------

// These package-level variables are the indirection points that let tests
// swap in fakes without build tags. In production, azure_credential.go
// assigns real implementations in an init() function.
var (
	newBrowserCredential         = defaultNewBrowserCredential
	newDeviceCodeCredential      = defaultNewDeviceCodeCredential
	newClientSecretCredential    = defaultNewClientSecretCredential
	newAzCLICredential           = defaultNewAzCLICredential
	newManagedIdentityCredential = defaultNewManagedIdentityCredential
)

// The defaults return an error indicating the Azure SDK is not linked.
// Production builds replace these via init() in azure_credential.go.

func defaultNewBrowserCredential(_ ProviderConfig) (Provider, error) {
	return nil, errors.AuthError("azure identity SDK not available (browser)", nil).
		WithSuggestion("Build with the azureidentity tag or ensure azure_credential.go is compiled.")
}

func defaultNewDeviceCodeCredential(_ ProviderConfig) (Provider, error) {
	return nil, errors.AuthError("azure identity SDK not available (device-code)", nil).
		WithSuggestion("Build with the azureidentity tag or ensure azure_credential.go is compiled.")
}

func defaultNewClientSecretCredential(_ ProviderConfig) (Provider, error) {
	return nil, errors.AuthError("azure identity SDK not available (client-credentials)", nil).
		WithSuggestion("Build with the azureidentity tag or ensure azure_credential.go is compiled.")
}

func defaultNewAzCLICredential(_ ProviderConfig) (Provider, error) {
	return nil, errors.AuthError("azure identity SDK not available (az-cli)", nil).
		WithSuggestion("Build with the azureidentity tag or ensure azure_credential.go is compiled.")
}

func defaultNewManagedIdentityCredential(_ ProviderConfig) (Provider, error) {
	return nil, errors.AuthError("azure identity SDK not available (managed-identity)", nil).
		WithSuggestion("Build with the azureidentity tag or ensure azure_credential.go is compiled.")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// methodNames returns sorted valid method names for error messages.
func methodNames() []string {
	return []string{
		MethodAzCLI,
		MethodBrowser,
		MethodClientCredential,
		MethodDeviceCode,
		MethodManagedIdentity,
	}
}
