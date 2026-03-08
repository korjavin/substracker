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
	mu            sync.RWMutex
	sessionCookie string
	baseURL       string
	client        *http.Client
}

func NewZAIProvider() *ZAIProvider {
	return &ZAIProvider{
		baseURL: "https://z.ai",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *ZAIProvider) Name() string {
	return "Z.ai"
}

func (p *ZAIProvider) Login(ctx context.Context, credentials map[string]string) error {
	cookie, ok := credentials["session_cookie"]
	if !ok || cookie == "" {
		return fmt.Errorf("session_cookie is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionCookie = cookie

	return nil
}

type zaiUsageResponse struct {
	Success *bool  `json:"success,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Code    int    `json:"code,omitempty"`
	Usage   *struct {
		Current int64  `json:"current"`
		Limit   int64  `json:"limit"`
		ResetAt string `json:"reset_at"`
	} `json:"usage,omitempty"`
}

func (p *ZAIProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	p.mu.RLock()
	cookie := p.sessionCookie
	baseURL := p.baseURL
	client := p.client
	p.mu.RUnlock()

	if cookie == "" {
		return nil, provider.ErrUnauthorized
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Cookie", fmt.Sprintf("session_cookie=%s", cookie))
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
		return nil, fmt.Errorf("api error: %s (code: %d)", usageResp.Msg, usageResp.Code)
	}

	if usageResp.Usage == nil {
		return nil, fmt.Errorf("missing usage data in response")
	}

	var resetDate time.Time
	if usageResp.Usage.ResetAt != "" {
		resetDate, _ = time.Parse(time.RFC3339, usageResp.Usage.ResetAt)
	}

	info := &provider.UsageInfo{
		ResetDate:           resetDate,
		CurrentUsageSeconds: usageResp.Usage.Current,
		TotalLimitSeconds:   usageResp.Usage.Limit,
		IsBlocked:           usageResp.Usage.Current >= usageResp.Usage.Limit && usageResp.Usage.Limit > 0,
	}

	return info, nil
}
