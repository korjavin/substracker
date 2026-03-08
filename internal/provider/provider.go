package provider

import (
	"context"
	"errors"
	"time"
)

// ErrUnauthorized is returned when the provider session is invalid or expired.
var ErrUnauthorized = errors.New("unauthorized: relogin required")

// UsageInfo represents the usage information retrieved from a provider.
type UsageInfo struct {
	// ResetDate is the date when the current usage limits reset.
	ResetDate time.Time
	// CurrentUsageSeconds is the current usage in seconds.
	CurrentUsageSeconds int64
	// TotalLimitSeconds is the total limit in seconds.
	TotalLimitSeconds int64
	// IsBlocked is true if the quota is currently blocked.
	IsBlocked bool
}

// Provider defines the interface for different service providers (Claude, OpenAI, Z.ai, etc.).
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Login authenticates with the provider using the provided credentials.
	Login(ctx context.Context, credentials map[string]string) error

	// FetchUsageInfo retrieves the current usage information from the provider.
	// It should be called after a successful Login.
	FetchUsageInfo(ctx context.Context) (*UsageInfo, error)
}
