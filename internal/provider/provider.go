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

	// Additional usage percentages and reset dates
	// Use pointers for floats so 0 is not omitted by omitempty
	SessionUsagePct *float64   `json:"session_usage_pct,omitempty"`
	SessionResetsAt time.Time  `json:"session_resets_at,omitempty"`
	WeeklyUsagePct  *float64   `json:"weekly_usage_pct,omitempty"`
	WeeklyResetsAt  time.Time  `json:"weekly_resets_at,omitempty"`
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
