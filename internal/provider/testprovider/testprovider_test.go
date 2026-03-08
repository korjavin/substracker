package testprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

func TestTestProvider_Name(t *testing.T) {
	p := NewTestProvider()
	if p.Name() != "TestProvider" {
		t.Errorf("expected TestProvider, got %s", p.Name())
	}
}

func TestTestProvider_FetchUsageInfo(t *testing.T) {
	p := NewTestProvider()
	ctx := context.Background()

	// Test case: Missing credentials
	_, err := p.FetchUsageInfo(ctx, nil)
	if !errors.Is(err, provider.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// Test case: Empty credentials
	_, err = p.FetchUsageInfo(ctx, map[string]string{})
	if !errors.Is(err, provider.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// Test case: Valid credentials
	usageInfo, err := p.FetchUsageInfo(ctx, map[string]string{"token": "dummy_token"})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if usageInfo == nil {
		t.Errorf("expected non-nil usageInfo")
	} else {
		// Check if reset date is roughly 5 days from now
		expectedDate := time.Now().Add(5 * 24 * time.Hour)
		diff := usageInfo.ResetDate.Sub(expectedDate)
		// Allow 1 second difference
		if diff > time.Second || diff < -time.Second {
			t.Errorf("expected reset date around %v, got %v", expectedDate, usageInfo.ResetDate)
		}
	}
}
