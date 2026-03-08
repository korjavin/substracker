package googleoneprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// GoogleOneProvider implements the Provider interface for Google One.
type GoogleOneProvider struct {
	mu         sync.RWMutex
	baseURL    string // Can be overridden for testing
	httpClient *http.Client
}

// NewGoogleOneProvider creates a new instance of GoogleOneProvider.
func NewGoogleOneProvider() *GoogleOneProvider {
	return &GoogleOneProvider{
		baseURL: "https://one.google.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the name of the provider.
func (p *GoogleOneProvider) Name() string {
	return "Google One"
}

type billingPeriod struct {
	EndDate string `json:"end_date"`
}

type subscriptionInfo struct {
	BillingPeriod billingPeriod `json:"billing_period"`
}

// FetchUsageInfo retrieves Google One storage details.
func (p *GoogleOneProvider) FetchUsageInfo(ctx context.Context, credentials map[string]string) (*provider.UsageInfo, error) {
	sessionCookie, ok := credentials["session_cookie"]
	if !ok || sessionCookie == "" {
		return nil, fmt.Errorf("googleoneprovider: %w", provider.ErrUnauthorized)
	}

	p.mu.RLock()
	baseURL := p.baseURL
	client := p.httpClient
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/subscriptions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// The user may provide just the SID value, or a full cookie string "SID=...; HSID=...; SSID=..."
	// We ensure it starts with SID= if they just pasted the value.
	cookieString := sessionCookie
	if !strings.Contains(cookieString, "=") {
		cookieString = fmt.Sprintf("SID=%s", sessionCookie)
	}
	req.Header.Set("Cookie", cookieString)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscriptions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("googleoneprovider: %w", provider.ErrUnauthorized)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching subscriptions: %d", resp.StatusCode)
	}

	// Note: the exact Google One API endpoint and response shape will need to be verified
	// by inspecting network traffic on one.google.com.
	// Currently expecting something like {"billing_period": {"end_date": "2024-05-10T15:00:00Z"}}
	var sub subscriptionInfo
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("failed to decode subscriptions: %w", err)
	}

	if sub.BillingPeriod.EndDate == "" {
		return nil, errors.New("no end_date found in subscription info")
	}

	// Parse reset date
	resetDate, err := time.Parse(time.RFC3339, sub.BillingPeriod.EndDate)
	if err != nil {
		// Fallback parsing for just date
		resetDate, err = time.Parse("2006-01-02", sub.BillingPeriod.EndDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end_date: %w", err)
		}
	}

	return &provider.UsageInfo{
		ResetDate: resetDate,
	}, nil
}
