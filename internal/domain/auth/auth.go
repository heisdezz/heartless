package auth

import (
	"context"
	"fmt"
	"net/http"
)

// AuthProvider is an interface for authentication providers.
// Each provider is responsible for applying its credentials to an outgoing request.
type AuthProvider interface {
	// ApplyCredentials applies the provider's credentials to the given request.
	ApplyCredentials(ctx context.Context, req *http.Request) error
	// Name returns the provider's name.
	Name() string
	// Type returns the authentication type.
	Type() string
}

// AuthResult represents the result of authentication application.
type AuthResult struct {
	// ProviderName is the name of the provider that applied credentials.
	ProviderName string
	// WasApplied indicates if credentials were successfully applied.
	WasApplied bool
	// Error contains any error that occurred during application.
	Error error
}

// NoAuthProvider provides no authentication.
type NoAuthProvider struct{}

// ApplyCredentials applies no credentials to the request.
func (p *NoAuthProvider) ApplyCredentials(ctx context.Context, req *http.Request) error {
	return nil
}

// Name returns the provider name.
func (p *NoAuthProvider) Name() string {
	return "none"
}

// Type returns the auth type.
func (p *NoAuthProvider) Type() string {
	return "none"
}

// BasicAuthProvider provides basic authentication.
type BasicAuthProvider struct {
	Username string
	Password string
}

// ApplyCredentials applies basic auth credentials to the request.
func (p *BasicAuthProvider) ApplyCredentials(ctx context.Context, req *http.Request) error {
	req.SetBasicAuth(p.Username, p.Password)
	return nil
}

// Name returns the provider name.
func (p *BasicAuthProvider) Name() string {
	return "basic"
}

// Type returns the auth type.
func (p *BasicAuthProvider) Type() string {
	return "basic"
}

// BearerAuthProvider provides bearer token authentication.
type BearerAuthProvider struct {
	Token string
}

// ApplyCredentials applies bearer token credentials to the request.
func (p *BearerAuthProvider) ApplyCredentials(ctx context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+p.Token)
	return nil
}

// Name returns the provider name.
func (p *BearerAuthProvider) Name() string {
	return "bearer"
}

// Type returns the auth type.
func (p *BearerAuthProvider) Type() string {
	return "bearer"
}

// APIKeyProvider provides API key authentication.
type APIKeyProvider struct {
	Key      string
	Header   string // e.g., "X-API-Key"
	Prefix   string // e.g., "ApiKey "
}

// ApplyCredentials applies API key credentials to the request.
func (p *APIKeyProvider) ApplyCredentials(ctx context.Context, req *http.Request) error {
	if p.Prefix != "" {
		req.Header.Set(p.Header, p.Prefix+p.Key)
	} else {
		req.Header.Set(p.Header, p.Key)
	}
	return nil
}

// Name returns the provider name.
func (p *APIKeyProvider) Name() string {
	return "api_key"
}

// Type returns the auth type.
func (p *APIKeyProvider) Type() string {
	return "api_key"
}

// AuthProviderManager manages authentication providers.
// Services depend on this manager to apply credentials to requests.
type AuthProviderManager interface {
	// GetProvider returns the auth provider for the given type and scope.
	GetProvider(providerType, scope string) (AuthProvider, error)
	// ListProviders returns all available providers for a given scope.
	ListProviders(scope string) map[string]AuthProvider
	// ApplyProviderCredentials applies the credentials from the selected provider.
	ApplyProviderCredentials(ctx context.Context, req *http.Request, providerType, scope string) error
	// RegisterProvider registers a new auth provider with the given scope.
	RegisterProvider(ctx context.Context, scope string, provider AuthProvider) error
}

// authProviderManager is the concrete implementation of AuthProviderManager.
type authProviderManager struct {
	providers map[string]map[string]AuthProvider // scope -> type -> provider
}

// NewAuthProviderManager creates a new auth provider manager.
func NewAuthProviderManager() *authProviderManager {
	return &authProviderManager{
		providers: make(map[string]map[string]AuthProvider),
	}
}

// GetProvider returns the auth provider for the given type and scope.
func (m *authProviderManager) GetProvider(providerType, scope string) (AuthProvider, error) {
	if scopeProviders, ok := m.providers[scope]; ok {
		if provider, ok := scopeProviders[providerType]; ok {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("auth provider %s not found in scope %s", providerType, scope)
}

// ListProviders returns all available providers for a given scope.
func (m *authProviderManager) ListProviders(scope string) map[string]AuthProvider {
	if scopeProviders, ok := m.providers[scope]; ok {
		return scopeProviders
	}
	return make(map[string]AuthProvider)
}

// ApplyProviderCredentials applies the credentials from the selected provider.
func (m *authProviderManager) ApplyProviderCredentials(ctx context.Context, req *http.Request, providerType, scope string) error {
	provider, err := m.GetProvider(providerType, scope)
	if err != nil {
		return err
	}
	return provider.ApplyCredentials(ctx, req)
}

// RegisterProvider registers a new auth provider with the given scope.
func (m *authProviderManager) RegisterProvider(ctx context.Context, scope string, provider AuthProvider) error {
	m.providers[scope] = map[string]AuthProvider{
		provider.Type(): provider,
	}
	return nil
}