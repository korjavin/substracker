package zaiprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

func TestZAIProvider_Name(t *testing.T) {
	p := NewZAIProvider()
	if p.Name() != "Z.ai" {
		t.Errorf("expected Z.ai, got %s", p.Name())
	}
}

func TestZAIProvider_Login(t *testing.T) {
	p := NewZAIProvider()
	ctx := context.Background()

	err := p.Login(ctx, nil)
	if err == nil {
		t.Errorf("expected error for nil credentials")
	}

	err = p.Login(ctx, map[string]string{})
	if err == nil {
		t.Errorf("expected error for empty credentials")
	}

	err = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	p.mu.RLock()
	if p.sessionCookie != "abc12345" {
		t.Errorf("expected abc12345, got %s", p.sessionCookie)
	}
	p.mu.RUnlock()
}

func TestZAIProvider_FetchUsageInfo(t *testing.T) {
	mockResponse := `{"data": {"current": 500, "limit": 1000, "reset_at": "2023-10-27T10:00:00Z"}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/monitor/usage/quota/limit" {
			t.Errorf("expected path /api/monitor/usage/quota/limit, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer abc12345" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	p := NewZAIProvider()
	p.baseURL = ts.URL
	ctx := context.Background()

	// Fetch without login
	_, err := p.FetchUsageInfo(ctx)
	if err != provider.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// Login and fetch
	err = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}

	info, err := p.FetchUsageInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}

	if info.CurrentUsageSeconds != 500 {
		t.Errorf("expected current usage 500, got %d", info.CurrentUsageSeconds)
	}
	if info.TotalLimitSeconds != 1000 {
		t.Errorf("expected total limit 1000, got %d", info.TotalLimitSeconds)
	}
	if info.IsBlocked {
		t.Errorf("expected not blocked")
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2023-10-27T10:00:00Z")
	if !info.ResetDate.Equal(expectedTime) {
		t.Errorf("expected reset date %v, got %v", expectedTime, info.ResetDate)
	}
}

func TestZAIProvider_FetchUsageInfo_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	p := NewZAIProvider()
	p.baseURL = ts.URL
	ctx := context.Background()
	_ = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})

	_, err := p.FetchUsageInfo(ctx)
	if err != provider.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestZAIProvider_FetchUsageInfo_APIError(t *testing.T) {
	mockResponse := `{"code":500,"msg":"404 NOT_FOUND","success":false}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	p := NewZAIProvider()
	p.baseURL = ts.URL
	ctx := context.Background()
	_ = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})

	_, err := p.FetchUsageInfo(ctx)
	if err == nil {
		t.Errorf("expected error for 200 API error response, got nil")
	} else if err.Error() != "api error: 404 NOT_FOUND (code: 500)" {
		t.Errorf("expected specific api error message, got %v", err)
	}
}

func TestZAIProvider_FetchUsageInfo_MissingUsage(t *testing.T) {
	mockResponse := `{"success":true,"other_data":"present"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	p := NewZAIProvider()
	p.baseURL = ts.URL
	ctx := context.Background()
	_ = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})

	_, err := p.FetchUsageInfo(ctx)
	if err == nil {
		t.Errorf("expected error for missing usage data, got nil")
	} else if err.Error() != "missing usage data in response" {
		t.Errorf("expected missing usage error, got %v", err)
	}
}

func TestZAIProvider_FetchUsageInfo_Malformed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{malformed json`))
	}))
	defer ts.Close()

	p := NewZAIProvider()
	p.baseURL = ts.URL
	ctx := context.Background()
	_ = p.Login(ctx, map[string]string{"session_cookie": "abc12345"})

	_, err := p.FetchUsageInfo(ctx)
	if err == nil {
		t.Errorf("expected error for malformed json")
	}
}
