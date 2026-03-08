package claudeprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

import "sync"

// ClaudeProvider implements the Provider interface for Claude.
type ClaudeProvider struct {
	mu         sync.RWMutex
	sessionKey string
	baseURL    string // Can be overridden for testing
	httpClient *http.Client
}

// NewClaudeProvider creates a new instance of ClaudeProvider.
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{
		baseURL:    "https://claude.ai/api",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the name of the provider.
func (p *ClaudeProvider) Name() string {
	return "Claude"
}

// Login authenticates with Claude by storing the session_key.
func (p *ClaudeProvider) Login(ctx context.Context, credentials map[string]string) error {
	sessionKey, ok := credentials["session_key"]
	if !ok || sessionKey == "" {
		return errors.New("missing or empty session_key")
	}

	p.mu.Lock()
	p.sessionKey = sessionKey
	p.mu.Unlock()
	return nil
}

type organization struct {
	UUID string `json:"uuid"`
}

type billingInfo struct {
	BillingPeriod struct {
		EndDate string `json:"end_date"`
	} `json:"billing_period"`
}

// organizationInfo is returned by Claude's API with usage limits
type organizationInfo struct {
	ActiveFlags []string `json:"active_flags"` // can contain "usage_limit_exceeded"
}

// FetchUsageInfo retrieves the current usage information from Claude.
func (p *ClaudeProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	p.mu.RLock()
	sessionKey := p.sessionKey
	p.mu.RUnlock()

	if sessionKey == "" {
		return nil, fmt.Errorf("claudeprovider: %w", provider.ErrUnauthorized)
	}

	// 1. Fetch organizations
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create organizations request: %w", err)
	}

	req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", sessionKey))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("claudeprovider: %w", provider.ErrUnauthorized)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching organizations: %d", resp.StatusCode)
	}

	var orgs []organization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	if len(orgs) == 0 {
		return nil, errors.New("no organizations found")
	}

	orgID := orgs[0].UUID

	// 2. Fetch usage/billing info for the first organization
	billingReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/organizations/%s/billing_info", p.baseURL, orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing request: %w", err)
	}

	billingReq.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", sessionKey))
	billingReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	billingResp, err := p.httpClient.Do(billingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch billing info: %w", err)
	}
	defer billingResp.Body.Close()

	if billingResp.StatusCode == http.StatusUnauthorized || billingResp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("claudeprovider: %w", provider.ErrUnauthorized)
	}

	if billingResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching billing info: %d", billingResp.StatusCode)
	}

	var billing billingInfo
	if err := json.NewDecoder(billingResp.Body).Decode(&billing); err != nil {
		return nil, fmt.Errorf("failed to decode billing info: %w", err)
	}

	// Parse reset date
	var resetDate time.Time
	if billing.BillingPeriod.EndDate != "" {
		// Claude often returns ISO8601 strings
		resetDate, err = time.Parse(time.RFC3339, billing.BillingPeriod.EndDate)
		if err != nil {
			// Fallback parsing for just date
			resetDate, err = time.Parse("2006-01-02", billing.BillingPeriod.EndDate)
			if err != nil {
				return nil, fmt.Errorf("failed to parse end_date: %w", err)
			}
		}
	} else {
		return nil, errors.New("no end_date found in billing info")
	}

	// 3. Check if the organization has "usage_limit_exceeded" flag
	orgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/organizations/%s", p.baseURL, orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create org info request: %w", err)
	}

	orgReq.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", sessionKey))
	orgReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	orgResp, err := p.httpClient.Do(orgReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch org info: %w", err)
	}
	defer orgResp.Body.Close()

	if orgResp.StatusCode == http.StatusUnauthorized || orgResp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("claudeprovider: %w", provider.ErrUnauthorized)
	}

	if orgResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching org info: %d", orgResp.StatusCode)
	}

	var isBlocked bool
	var oInfo organizationInfo
	if err := json.NewDecoder(orgResp.Body).Decode(&oInfo); err != nil {
		return nil, fmt.Errorf("failed to decode org info: %w", err)
	}

	for _, flag := range oInfo.ActiveFlags {
		if flag == "usage_limit_exceeded" || flag == "message_limit_exceeded" {
			isBlocked = true
			break
		}
	}

	return &provider.UsageInfo{
		ResetDate:           resetDate,
		CurrentUsageSeconds: 0, // Claude doesn't easily expose this, we rely on flags
		TotalLimitSeconds:   0,
		IsBlocked:           isBlocked,
	}, nil
}
