package zaiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

type ZAIProvider struct {
	mu      sync.RWMutex
	baseURL string
	client  *http.Client
}

const zaiQuotaLimitPath = "/api/monitor/usage/quota/limit"

func NewZAIProvider() *ZAIProvider {
	return &ZAIProvider{
		baseURL: "https://api.z.ai",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *ZAIProvider) Name() string {
	return "Z.ai"
}

type zaiUsageResponse struct {
	Success *bool  `json:"success,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Code    int    `json:"code,omitempty"`
	Data    *struct {
		Current int64  `json:"current"`
		Limit   int64  `json:"limit"`
		ResetAt string `json:"reset_at"`
	} `json:"data,omitempty"`
}

func (p *ZAIProvider) FetchUsageInfo(ctx context.Context, credentials map[string]string) (*provider.UsageInfo, error) {
	cookie := credentials["session_cookie"]

	p.mu.RLock()
	baseURL := p.baseURL
	client := p.client
	p.mu.RUnlock()

	if cookie == "" {
		return nil, provider.ErrUnauthorized
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+zaiQuotaLimitPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Cookie", fmt.Sprintf("session_cookie=%s", cookie))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cookie))
	req.Header.Set("Origin", "https://z.ai")
	req.Header.Set("Referer", "https://z.ai/")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, provider.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var usageResp zaiUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if usageResp.Success != nil && !*usageResp.Success {
		if usageResp.Code == 401 || usageResp.Code == 1001 {
			return nil, provider.ErrUnauthorized
		}
		return nil, fmt.Errorf("api error: %s (code: %d)", usageResp.Msg, usageResp.Code)
	}

	if usageResp.Data == nil {
		return nil, fmt.Errorf("missing usage data in response")
	}

	var resetDate time.Time
	if usageResp.Data.ResetAt != "" {
		resetDate, _ = time.Parse(time.RFC3339, usageResp.Data.ResetAt)
	}

	info := &provider.UsageInfo{
		ResetDate:           resetDate,
		CurrentUsageSeconds: usageResp.Data.Current,
		TotalLimitSeconds:   usageResp.Data.Limit,
		IsBlocked:           usageResp.Data.Current >= usageResp.Data.Limit && usageResp.Data.Limit > 0,
	}

	return info, nil
}
