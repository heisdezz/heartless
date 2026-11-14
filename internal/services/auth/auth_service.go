package auth_service

import (
	"context"
	"net/http"

	"heartless/internal/domain/auth"
)

// AuthService handles authentication-related business logic.
// It depends on the provider manager interface, keeping auth logic separate from persistence.
type AuthService interface {
	// RegisterProvider registers an auth provider with the given scope.
	RegisterProvider(ctx context.Context, scope string, provider auth.AuthProvider) error
	// GetProvider returns the auth provider for the given type and scope.
	GetProvider(ctx context.Context, providerType, scope string) (auth.AuthProvider, error)
	// ListProviders returns all available providers for a given scope.
	ListProviders(ctx context.Context, scope string) map[string]auth.AuthProvider
	// ApplyProviderCredentials applies the credentials from the selected provider to a request.
	ApplyProviderCredentials(ctx context.Context, req *http.Request, providerType, scope string) error
	// AuthenticateRequest authenticates a request using the configured auth settings.
	AuthenticateRequest(ctx context.Context, req *http.Request, providerType, scope string) error
}

// authService is the concrete implementation of AuthService.
// The AuthProviderManager is provided at startup (in-memory or SQLite-backed).
type authService struct {
	manager auth.AuthProviderManager
}

// NewAuthService creates a new auth service with the given provider manager.
func NewAuthService(manager auth.AuthProviderManager) AuthService {
	return &authService{manager: manager}
}

// RegisterProvider registers an auth provider with the given scope.
func (s *authService) RegisterProvider(ctx context.Context, scope string, provider auth.AuthProvider) error {
	return s.manager.RegisterProvider(ctx, scope, provider)
}

// GetProvider returns the auth provider for the given type and scope.
func (s *authService) GetProvider(ctx context.Context, providerType, scope string) (auth.AuthProvider, error) {
	return s.manager.GetProvider(providerType, scope)
}

// ListProviders returns all available providers for a given scope.
func (s *authService) ListProviders(ctx context.Context, scope string) map[string]auth.AuthProvider {
	return s.manager.ListProviders(scope)
}

// ApplyProviderCredentials applies the credentials from the selected provider to a request.
func (s *authService) ApplyProviderCredentials(ctx context.Context, req *http.Request, providerType, scope string) error {
	return s.manager.ApplyProviderCredentials(ctx, req, providerType, scope)
}

// AuthenticateRequest authenticates a request using the configured auth settings.
func (s *authService) AuthenticateRequest(ctx context.Context, req *http.Request, providerType, scope string) error {
	return s.ApplyProviderCredentials(ctx, req, providerType, scope)
}
