// API Endpoint Documented for chatgpt.com usage scraping (Task 1)
// Endpoint URL: https://chatgpt.com/backend-api/wham/usage
// HTTP Method: GET
// Required Headers: Accept: application/json, Authorization: Bearer <token>, Cookie: __Secure-next-auth.session-token=<token>
// Response JSON Schema includes plan_type, rate_limit (primary_window and secondary_window objects)
// fields: used_percent (Double), limit_window_seconds (Double), reset_after_seconds (Double), reset_at (TimeInterval)
package openaiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// OpenAIProvider implements the Provider interface for OpenAI/Codex.
type OpenAIProvider struct {
	mu         sync.RWMutex
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new instance of OpenAIProvider.
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: "https://chatgpt.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the name of the provider.
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

type rateLimitWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	LimitWindowSeconds *float64 `json:"limit_window_seconds"`
	ResetAfterSeconds  *float64 `json:"reset_after_seconds"`
	ResetAt            *float64 `json:"reset_at"`
}

type rateLimit struct {
	PrimaryWindow   *rateLimitWindow `json:"primary_window"`
	SecondaryWindow *rateLimitWindow `json:"secondary_window"`
}

type codexUsageResponse struct {
	PlanType  *string    `json:"plan_type"`
	RateLimit *rateLimit `json:"rate_limit"`
	Detail    *string    `json:"detail"` // Used for error messages like "Unauthorized"
}

// FetchUsageInfo retrieves the current usage information from OpenAI/chatgpt.com.
func (p *OpenAIProvider) FetchUsageInfo(ctx context.Context, credentials map[string]string) (*provider.UsageInfo, error) {
	sessionToken := credentials["session_token"]
	if sessionToken == "" {
		return nil, fmt.Errorf("openaiprovider: %w", provider.ErrUnauthorized)
	}

	p.mu.RLock()
	httpClient := p.httpClient
	baseURL := p.baseURL
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/backend-api/wham/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Authenticate using the session cookie and bearer token
	cookieString := fmt.Sprintf("__Secure-next-auth.session-token=%s", sessionToken)
	req.Header.Set("Cookie", cookieString)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sessionToken))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("openaiprovider: %w", provider.ErrUnauthorized)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching usage: %d", resp.StatusCode)
	}

	var usage codexUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, fmt.Errorf("failed to decode usage: %w", err)
	}

	if usage.Detail != nil && *usage.Detail == "Unauthorized" {
		return nil, fmt.Errorf("openaiprovider: %w", provider.ErrUnauthorized)
	}

	var resetDate time.Time
	var currentUsageSeconds int64
	var totalLimitSeconds int64
	var isBlocked bool

	if usage.RateLimit != nil {
		// Check both primary and secondary windows.
		// Use primary window for metrics if available.
		// If either window indicates blocking, set isBlocked to true.

		if usage.RateLimit.PrimaryWindow != nil {
			window := usage.RateLimit.PrimaryWindow

			if window.ResetAt != nil {
				resetDate = time.Unix(int64(*window.ResetAt), 0)
			} else if window.ResetAfterSeconds != nil {
				resetDate = time.Now().Add(time.Duration(*window.ResetAfterSeconds) * time.Second)
			}

			if window.LimitWindowSeconds != nil {
				totalLimitSeconds = int64(*window.LimitWindowSeconds)
			}

			if window.UsedPercent != nil && totalLimitSeconds > 0 {
				currentUsageSeconds = int64(*window.UsedPercent * float64(totalLimitSeconds))
				if *window.UsedPercent >= 1.0 {
					isBlocked = true
				}
			}
		}

		if usage.RateLimit.SecondaryWindow != nil {
			window := usage.RateLimit.SecondaryWindow

			// If primary didn't provide reset date, try secondary
			if resetDate.IsZero() {
				if window.ResetAt != nil {
					resetDate = time.Unix(int64(*window.ResetAt), 0)
				} else if window.ResetAfterSeconds != nil {
					resetDate = time.Now().Add(time.Duration(*window.ResetAfterSeconds) * time.Second)
				}
			}

			// If primary didn't provide limits, try secondary
			if totalLimitSeconds == 0 && window.LimitWindowSeconds != nil {
				totalLimitSeconds = int64(*window.LimitWindowSeconds)
			}

			// If primary didn't provide usage, try secondary
			if currentUsageSeconds == 0 && window.UsedPercent != nil && totalLimitSeconds > 0 {
				currentUsageSeconds = int64(*window.UsedPercent * float64(totalLimitSeconds))
			}

			// Block if secondary is over limit, regardless of primary
			if window.UsedPercent != nil && *window.UsedPercent >= 1.0 {
				isBlocked = true
			}
		}
	}

	if resetDate.IsZero() {
		// Fallback to end of the current month if no reset date is provided
		now := time.Now()
		nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		resetDate = nextMonth.Add(-24 * time.Hour)
	}

	return &provider.UsageInfo{
		ResetDate:           resetDate,
		CurrentUsageSeconds: currentUsageSeconds,
		TotalLimitSeconds:   totalLimitSeconds,
		IsBlocked:           isBlocked,
	}, nil
}
