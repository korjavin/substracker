package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/repository"
	_ "modernc.org/sqlite"
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
	return &provider.UsageInfo{
		ResetDate:           time.Now(),
		CurrentUsageSeconds: 3600,
		TotalLimitSeconds:   7200,
		IsBlocked:           true,
	}, nil
}

func setupTestDB(t *testing.T) *repository.Queries {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE provider_credentials (
			provider_name TEXT NOT NULL,
			credential_key TEXT NOT NULL,
			credential_value TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider_name, credential_key)
		);

		CREATE TABLE provider_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_name TEXT UNIQUE NOT NULL,
			current_usage_seconds INTEGER NOT NULL DEFAULT 0,
			total_limit_seconds INTEGER NOT NULL DEFAULT 0,
			is_blocked INTEGER NOT NULL DEFAULT 0,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return repository.New(db)
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
	repo := setupTestDB(t)
	m := &mockProvider{}
	h := &Handler{repo: repo, claudeProvider: m}

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

	// verify that the credential was persisted to DB
	ctx := context.Background()
	savedKey, err := repo.GetProviderCredential(ctx, m.Name(), "session_key")
	if err != nil {
		t.Errorf("expected to find credential in DB, got error: %v", err)
	}
	if savedKey != "test_key" {
		t.Errorf("expected saved credential to be 'test_key', got '%s'", savedKey)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/claude/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.claudeLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}

	// Test persistence failure
	dbFail, _ := sql.Open("sqlite", ":memory:")
	dbFail.Close() // Force operations to fail
	repoFail := repository.New(dbFail)
	hFail := &Handler{repo: repoFail, claudeProvider: &mockProvider{}}
	reqFail := httptest.NewRequest(http.MethodPost, "/api/providers/claude/login", bytes.NewBuffer([]byte(`{"session_key": "fail_key"}`)))
	rrFail := httptest.NewRecorder()
	hFail.claudeLogin(rrFail, reqFail)

	if rrFail.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 when DB write fails, got %d", rrFail.Code)
	}
}

func TestClaudeUsage(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockProvider{sessionKey: "valid_key"}
	h := &Handler{repo: repo, claudeProvider: m}

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

	// Test cachedUsage
	reqCached := httptest.NewRequest(http.MethodGet, "/api/providers/usage/cached", nil)
	rrCached := httptest.NewRecorder()
	h.cachedUsage(rrCached, reqCached)

	if rrCached.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rrCached.Code)
	}

	var resCached []repository.ProviderUsage
	if err := json.NewDecoder(rrCached.Body).Decode(&resCached); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resCached) != 1 {
		t.Fatalf("expected 1 cached usage, got %d", len(resCached))
	}

	if resCached[0].ProviderName != "MockClaude" {
		t.Errorf("expected 'MockClaude', got '%s'", resCached[0].ProviderName)
	}
	if resCached[0].CurrentUsageSeconds != 3600 {
		t.Errorf("expected 3600, got %d", resCached[0].CurrentUsageSeconds)
	}
	if !resCached[0].IsBlocked {
		t.Errorf("expected IsBlocked to be true")
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
	repo := setupTestDB(t)
	m := &mockGoogleOneProvider{}
	h := &Handler{repo: repo, googleOneProvider: m}

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

	// verify credential persistence
	savedCookie, err := repo.GetProviderCredential(context.Background(), m.Name(), "session_cookie")
	if err != nil {
		t.Errorf("expected to find credential in DB, got error: %v", err)
	}
	if savedCookie != "test_cookie" {
		t.Errorf("expected saved credential to be 'test_cookie', got '%s'", savedCookie)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/googleone/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.googleOneLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}

	// Test persistence failure
	dbFail, _ := sql.Open("sqlite", ":memory:")
	dbFail.Close() // Force operations to fail
	repoFail := repository.New(dbFail)
	hFail := &Handler{repo: repoFail, googleOneProvider: &mockGoogleOneProvider{}}
	reqFail := httptest.NewRequest(http.MethodPost, "/api/providers/googleone/login", bytes.NewBuffer([]byte(`{"session_cookie": "fail_cookie"}`)))
	rrFail := httptest.NewRecorder()
	hFail.googleOneLogin(rrFail, reqFail)

	if rrFail.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 when DB write fails, got %d", rrFail.Code)
	}
}

// mockOpenAIProvider implements provider.Provider for testing API endpoints
type mockOpenAIProvider struct {
	sessionToken string
	shouldFail   bool
}

func (m *mockOpenAIProvider) Name() string {
	return "MockOpenAI"
}

func (m *mockOpenAIProvider) Login(ctx context.Context, credentials map[string]string) error {
	if m.shouldFail {
		return provider.ErrUnauthorized
	}
	m.sessionToken = credentials["session_token"]
	return nil
}

func (m *mockOpenAIProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if m.shouldFail {
		return nil, provider.ErrUnauthorized
	}
	if m.sessionToken == "" {
		return nil, provider.ErrUnauthorized
	}
	return &provider.UsageInfo{ResetDate: time.Now()}, nil
}

func TestOpenAILoginInfo(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/providers/openai/login-info", nil)
	rr := httptest.NewRecorder()

	h.openaiLoginInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["url"] != "https://platform.openai.com/" {
		t.Errorf("expected url 'https://platform.openai.com/', got '%s'", res["url"])
	}
}

func TestOpenAILogin(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockOpenAIProvider{}
	h := &Handler{repo: repo, openaiProvider: m}

	body := []byte(`{"session_token": "test_token"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/openai/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.openaiLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if m.sessionToken != "test_token" {
		t.Errorf("expected mock provider to store 'test_token', got '%s'", m.sessionToken)
	}

	// verify credential persistence
	savedToken, err := repo.GetProviderCredential(context.Background(), m.Name(), "session_token")
	if err != nil {
		t.Errorf("expected to find credential in DB, got error: %v", err)
	}
	if savedToken != "test_token" {
		t.Errorf("expected saved credential to be 'test_token', got '%s'", savedToken)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/openai/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.openaiLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}
}

func TestOpenAIUsage(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockOpenAIProvider{sessionToken: "valid_token"}
	h := &Handler{repo: repo, openaiProvider: m}

	// Test success
	req := httptest.NewRequest(http.MethodGet, "/api/providers/openai/usage", nil)
	rr := httptest.NewRecorder()
	h.openaiUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	usage, err := repo.GetProviderUsage(context.Background(), "MockOpenAI")
	if err != nil {
		t.Fatalf("failed to get cached usage: %v", err)
	}
	if usage.ProviderName != "MockOpenAI" {
		t.Errorf("expected cached usage for 'MockOpenAI', got '%s'", usage.ProviderName)
	}

	// Test unauthorized
	mUnauthorized := &mockOpenAIProvider{shouldFail: true}
	hUnauthorized := &Handler{openaiProvider: mUnauthorized}

	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/providers/openai/usage", nil)
	rrUnauth := httptest.NewRecorder()
	hUnauthorized.openaiUsage(rrUnauth, reqUnauth)

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

func TestGoogleOneUsage(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockGoogleOneProvider{sessionCookie: "valid_cookie"}
	h := &Handler{repo: repo, googleOneProvider: m}

	// Test success
	req := httptest.NewRequest(http.MethodGet, "/api/providers/googleone/usage", nil)
	rr := httptest.NewRecorder()
	h.googleOneUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	usage, err := repo.GetProviderUsage(context.Background(), "MockGoogleOne")
	if err != nil {
		t.Fatalf("failed to get cached usage: %v", err)
	}
	if usage.ProviderName != "MockGoogleOne" {
		t.Errorf("expected cached usage for 'MockGoogleOne', got '%s'", usage.ProviderName)
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

// mockZAIProvider implements provider.Provider for testing API endpoints
type mockZAIProvider struct {
	sessionCookie string
	shouldFail    bool
}

func (m *mockZAIProvider) Name() string {
	return "MockZAI"
}

func (m *mockZAIProvider) Login(ctx context.Context, credentials map[string]string) error {
	if m.shouldFail {
		return provider.ErrUnauthorized
	}
	m.sessionCookie = credentials["session_cookie"]
	return nil
}

func (m *mockZAIProvider) FetchUsageInfo(ctx context.Context) (*provider.UsageInfo, error) {
	if m.shouldFail {
		return nil, provider.ErrUnauthorized
	}
	if m.sessionCookie == "" {
		return nil, provider.ErrUnauthorized
	}
	return &provider.UsageInfo{
		ResetDate:           time.Now(),
		CurrentUsageSeconds: 100,
		TotalLimitSeconds:   500,
		IsBlocked:           false,
	}, nil
}

func TestZAILoginInfo(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/providers/zai/login-info", nil)
	rr := httptest.NewRecorder()

	h.zaiLoginInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["url"] != "https://z.ai/" {
		t.Errorf("expected url 'https://z.ai/', got '%s'", res["url"])
	}
}

func TestZAILogin(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockZAIProvider{}
	h := &Handler{repo: repo, zaiProvider: m}

	body := []byte(`{"session_cookie": "test_cookie"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/zai/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.zaiLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if m.sessionCookie != "test_cookie" {
		t.Errorf("expected mock provider to store 'test_cookie', got '%s'", m.sessionCookie)
	}

	cred, err := repo.GetProviderCredential(context.Background(), "MockZAI", "session_cookie")
	if err != nil {
		t.Fatalf("failed to retrieve credential from db: %v", err)
	}
	if cred != "test_cookie" {
		t.Errorf("expected stored credential 'test_cookie', got '%s'", cred)
	}

	// Test invalid body
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/providers/zai/login", bytes.NewBuffer([]byte(`{invalid_json}`)))
	rrInvalid := httptest.NewRecorder()
	h.zaiLogin(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", rrInvalid.Code)
	}
}

func TestZAIUsage(t *testing.T) {
	repo := setupTestDB(t)
	m := &mockZAIProvider{sessionCookie: "valid_cookie"}
	h := &Handler{repo: repo, zaiProvider: m}

	// Test success
	req := httptest.NewRequest(http.MethodGet, "/api/providers/zai/usage", nil)
	rr := httptest.NewRecorder()
	h.zaiUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	usage, err := repo.GetProviderUsage(context.Background(), "MockZAI")
	if err != nil {
		t.Fatalf("failed to retrieve usage from db: %v", err)
	}
	if usage.CurrentUsageSeconds != 100 {
		t.Errorf("expected stored usage 100, got '%d'", usage.CurrentUsageSeconds)
	}

	// Test unauthorized
	mUnauthorized := &mockZAIProvider{shouldFail: true}
	hUnauthorized := &Handler{zaiProvider: mUnauthorized}

	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/providers/zai/usage", nil)
	rrUnauth := httptest.NewRecorder()
	hUnauthorized.zaiUsage(rrUnauth, reqUnauth)

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
