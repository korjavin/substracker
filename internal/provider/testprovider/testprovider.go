package testprovider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// TestProvider is a dummy implementation of the Provider interface for testing purposes.
type TestProvider struct {
	authenticated bool
	resetDate     time.Time
}

// NewTestProvider creates a new instance of TestProvider.
func NewTestProvider() *TestProvider {
	return &TestProvider{
		authenticated: false,
	}
}

// Name returns the name of the test provider.
func (p *TestProvider) Name() string {
	return "TestProvider"
}

// Login simulates a login process. It requires a "token" key in the credentials map.
func (p *TestProvider) Login(ctx context.Context, credentials map[string]string) error {
	token, ok := credentials["token"]
	if !ok || token == "" {
		return errors.New("missing or empty token")
	}

	// Simulate successful authentication
	p.authenticated = true
	// Simulate a reset date 5 days from now
	p.resetDate = time.Now().Add(5 * 24 * time.Hour)
	return nil
}

// FetchUsageInfo simulates fetching usage information. It requires the provider to be authenticated.
func (p *TestProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if !p.authenticated {
		return nil, fmt.Errorf("testprovider: %w", provider.ErrUnauthorized)
	}

	return &provider.UsageInfo{
		ResetDate: p.resetDate,
	}, nil
}
