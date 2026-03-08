package openaiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// OpenAIProvider implements the Provider interface for OpenAI/Codex.
type OpenAIProvider struct {
	mu           sync.RWMutex
	sessionToken string
	baseURL      string
	httpClient   *http.Client
}

// NewOpenAIProvider creates a new instance of OpenAIProvider.
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: "https://api.openai.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the name of the provider.
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// Login authenticates with OpenAI using the provided credentials.
func (p *OpenAIProvider) Login(ctx context.Context, credentials map[string]string) error {
	sessionToken, ok := credentials["session_token"]
	if !ok || sessionToken == "" {
		return errors.New("missing or empty session_token")
	}

	p.mu.Lock()
	p.sessionToken = sessionToken
	p.mu.Unlock()
	return nil
}

type subscriptionPlan struct {
	ID string `json:"id"`
}

type billingSubscription struct {
	Object         string           `json:"object"`
	HasPaymentInfo bool             `json:"has_payment_method"`
	Plan           subscriptionPlan `json:"plan"`
	AccessUntil    int64            `json:"access_until"` // Timestamp
}

// FetchUsageInfo retrieves the current usage information from OpenAI.
func (p *OpenAIProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	p.mu.RLock()
	sessionToken := p.sessionToken
	p.mu.RUnlock()

	if sessionToken == "" {
		return nil, fmt.Errorf("openaiprovider: %w", provider.ErrUnauthorized)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/dashboard/billing/subscription", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Authenticate using the session cookie
	cookieString := fmt.Sprintf("__Secure-next-auth.session-token=%s", sessionToken)
	req.Header.Set("Cookie", cookieString)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch billing subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("openaiprovider: %w", provider.ErrUnauthorized)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching subscription: %d", resp.StatusCode)
	}

	var sub billingSubscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return nil, fmt.Errorf("failed to decode subscription: %w", err)
	}

	var resetDate time.Time

	if sub.AccessUntil > 0 {
		resetDate = time.Unix(sub.AccessUntil, 0)
	} else {
		// If access_until is not present or 0, fallback to end of the current month
		now := time.Now()
		// Get last day of current month
		nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		resetDate = nextMonth.Add(-24 * time.Hour)
	}

	return &provider.UsageInfo{
		ResetDate: resetDate,
	}, nil
}
