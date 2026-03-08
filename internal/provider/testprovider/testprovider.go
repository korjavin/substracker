package testprovider

import (
	"context"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// TestProvider is a dummy implementation of the Provider interface for testing purposes.
type TestProvider struct {
	resetDate time.Time
}

// NewTestProvider creates a new instance of TestProvider.
func NewTestProvider() *TestProvider {
	return &TestProvider{}
}

// Name returns the name of the test provider.
func (p *TestProvider) Name() string {
	return "TestProvider"
}

// FetchUsageInfo simulates fetching usage information. It requires a "token" key in the credentials map.
func (p *TestProvider) FetchUsageInfo(ctx context.Context, credentials map[string]string) (*provider.UsageInfo, error) {
	token, ok := credentials["token"]
	if !ok || token == "" {
		return nil, provider.ErrUnauthorized
	}

	// Simulate a reset date 5 days from now
	p.resetDate = time.Now().Add(5 * 24 * time.Hour)

	return &provider.UsageInfo{
		ResetDate:           p.resetDate,
		CurrentUsageSeconds: 10800, // 3 hours
		TotalLimitSeconds:   18000, // 5 hours
		IsBlocked:           false,
	}, nil
}
