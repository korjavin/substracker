package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

// mockProvider implements provider.Provider for testing API endpoints
type mockProvider struct {
	sessionKey string
	shouldFail bool
}

func (m *mockProvider) Name() string {
	return "MockClaude"
}

func (m *mockProvider) Login(ctx context.Context, credentials map[string]string) error {
	if m.shouldFail {
		return provider.ErrUnauthorized
	}
	m.sessionKey = credentials["session_key"]
	return nil
}

func (m *mockProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if m.shouldFail {
		return nil, provider.ErrUnauthorized
	}
	if m.sessionKey == "" {
		return nil, provider.ErrUnauthorized
	}
	return &provider.UsageInfo{ResetDate: time.Now()}, nil
}

func TestClaudeLoginInfo(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/providers/claude/login-info", nil)
	rr := httptest.NewRecorder()

	h.claudeLoginInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["url"] != "https://claude.ai/" {
		t.Errorf("expected url 'https://claude.ai/', got '%s'", res["url"])
	}
}

func TestClaudeLogin(t *testing.T) {
	m := &mockProvider{}
	h := &Handler{claudeProvider: m}

	body := []byte(`{"session_key": "test_key"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/claude/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.claudeLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if m.sessionKey != "test_key" {
		t.Errorf("expected mock provider to store 'test_key', got '%s'", m.sessionKey)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/claude/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.claudeLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}
}

func TestClaudeUsage(t *testing.T) {
	m := &mockProvider{sessionKey: "valid_key"}
	h := &Handler{claudeProvider: m}

	// Test success
	req := httptest.NewRequest(http.MethodGet, "/api/providers/claude/usage", nil)
	rr := httptest.NewRecorder()
	h.claudeUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test unauthorized
	mUnauthorized := &mockProvider{shouldFail: true}
	hUnauthorized := &Handler{claudeProvider: mUnauthorized}

	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/providers/claude/usage", nil)
	rrUnauth := httptest.NewRecorder()
	hUnauthorized.claudeUsage(rrUnauth, reqUnauth)

	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rrUnauth.Code)
	}

	var resUnauth map[string]string
	if err := json.NewDecoder(rrUnauth.Body).Decode(&resUnauth); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resUnauth["error"] != "relogin_required" {
		t.Errorf("expected error 'relogin_required', got '%s'", resUnauth["error"])
	}
}

// mockGoogleOneProvider implements provider.Provider for testing API endpoints
type mockGoogleOneProvider struct {
	sessionCookie string
	shouldFail    bool
}

func (m *mockGoogleOneProvider) Name() string {
	return "MockGoogleOne"
}

func (m *mockGoogleOneProvider) Login(ctx context.Context, credentials map[string]string) error {
	if m.shouldFail {
		return provider.ErrUnauthorized
	}
	m.sessionCookie = credentials["session_cookie"]
	return nil
}

func (m *mockGoogleOneProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if m.shouldFail {
		return nil, provider.ErrUnauthorized
	}
	if m.sessionCookie == "" {
		return nil, provider.ErrUnauthorized
	}
	return &provider.UsageInfo{ResetDate: time.Now()}, nil
}

func TestGoogleOneLoginInfo(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/providers/googleone/login-info", nil)
	rr := httptest.NewRecorder()

	h.googleOneLoginInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["url"] != "https://one.google.com/" {
		t.Errorf("expected url 'https://one.google.com/', got '%s'", res["url"])
	}
}

func TestGoogleOneLogin(t *testing.T) {
	m := &mockGoogleOneProvider{}
	h := &Handler{googleOneProvider: m}

	body := []byte(`{"session_cookie": "test_cookie"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/googleone/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.googleOneLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if m.sessionCookie != "test_cookie" {
		t.Errorf("expected mock provider to store 'test_cookie', got '%s'", m.sessionCookie)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/googleone/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.googleOneLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}
}

func TestGoogleOneUsage(t *testing.T) {
	m := &mockGoogleOneProvider{sessionCookie: "valid_cookie"}
	h := &Handler{googleOneProvider: m}

	// Test success
	req := httptest.NewRequest(http.MethodGet, "/api/providers/googleone/usage", nil)
	rr := httptest.NewRecorder()
	h.googleOneUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test unauthorized
	mUnauthorized := &mockGoogleOneProvider{shouldFail: true}
	hUnauthorized := &Handler{googleOneProvider: mUnauthorized}

	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/providers/googleone/usage", nil)
	rrUnauth := httptest.NewRecorder()
	hUnauthorized.googleOneUsage(rrUnauth, reqUnauth)

	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rrUnauth.Code)
	}

	var resUnauth map[string]string
	if err := json.NewDecoder(rrUnauth.Body).Decode(&resUnauth); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resUnauth["error"] != "relogin_required" {
		t.Errorf("expected error 'relogin_required', got '%s'", resUnauth["error"])
	}
}
