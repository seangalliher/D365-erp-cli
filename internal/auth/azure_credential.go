package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func init() {
	// Wire up real Azure Identity credentials, replacing the stubs.
	newBrowserCredential = realNewBrowserCredential
	newDeviceCodeCredential = realNewDeviceCodeCredential
	newClientSecretCredential = realNewClientSecretCredential
	newAzCLICredential = realNewAzCLICredential
	newManagedIdentityCredential = realNewManagedIdentityCredential
}

// azureTokenProvider wraps any azcore.TokenCredential into our Provider interface.
type azureTokenProvider struct {
	cred   azcore.TokenCredential
	method string
}

func (a *azureTokenProvider) GetToken(ctx context.Context, scopes []string) (*TokenResult, error) {
	tok, err := a.cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}
	return &TokenResult{
		AccessToken: tok.Token,
		ExpiresOn:   tok.ExpiresOn.UTC(),
	}, nil
}

func (a *azureTokenProvider) Method() string {
	return a.method
}

func realNewBrowserCredential(cfg ProviderConfig) (Provider, error) {
	opts := &azidentity.InteractiveBrowserCredentialOptions{
		TenantID: cfg.TenantID,
		ClientID: cfg.ClientID,
	}
	if cfg.RedirectURL != "" {
		opts.RedirectURL = cfg.RedirectURL
	}
	cred, err := azidentity.NewInteractiveBrowserCredential(opts)
	if err != nil {
		return nil, fmt.Errorf("browser credential: %w", err)
	}
	return &azureTokenProvider{cred: cred, method: MethodBrowser}, nil
}

func realNewDeviceCodeCredential(cfg ProviderConfig) (Provider, error) {
	opts := &azidentity.DeviceCodeCredentialOptions{
		TenantID: cfg.TenantID,
		ClientID: cfg.ClientID,
	}
	if cfg.DeviceCodeHandler != nil {
		opts.UserPrompt = func(ctx context.Context, msg azidentity.DeviceCodeMessage) error {
			cfg.DeviceCodeHandler(ctx, msg.Message)
			return nil
		}
	} else {
		opts.UserPrompt = func(ctx context.Context, msg azidentity.DeviceCodeMessage) error {
			fmt.Fprintln(os.Stderr, msg.Message)
			return nil
		}
	}
	cred, err := azidentity.NewDeviceCodeCredential(opts)
	if err != nil {
		return nil, fmt.Errorf("device-code credential: %w", err)
	}
	return &azureTokenProvider{cred: cred, method: MethodDeviceCode}, nil
}

func realNewClientSecretCredential(cfg ProviderConfig) (Provider, error) {
	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("client-secret credential: %w", err)
	}
	return &azureTokenProvider{cred: cred, method: MethodClientCredential}, nil
}

func realNewAzCLICredential(_ ProviderConfig) (Provider, error) {
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf("az-cli credential: %w", err)
	}
	return &azureTokenProvider{cred: cred, method: MethodAzCLI}, nil
}

func realNewManagedIdentityCredential(cfg ProviderConfig) (Provider, error) {
	opts := &azidentity.ManagedIdentityCredentialOptions{}
	if cfg.ClientID != "" {
		opts.ID = azidentity.ClientID(cfg.ClientID)
	}
	cred, err := azidentity.NewManagedIdentityCredential(opts)
	if err != nil {
		return nil, fmt.Errorf("managed-identity credential: %w", err)
	}
	return &azureTokenProvider{cred: cred, method: MethodManagedIdentity}, nil
}
