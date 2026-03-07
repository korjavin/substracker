package claudeprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

func TestClaudeProvider_Name(t *testing.T) {
	p := NewClaudeProvider()
	if name := p.Name(); name != "Claude" {
		t.Errorf("expected Name to be 'Claude', got '%s'", name)
	}
}

func TestClaudeProvider_Login(t *testing.T) {
	p := NewClaudeProvider()
	ctx := context.Background()

	// Empty credentials
	err := p.Login(ctx, map[string]string{})
	if err == nil {
		t.Error("expected error for empty credentials, got nil")
	}

	// Missing session_key
	err = p.Login(ctx, map[string]string{"other_key": "value"})
	if err == nil {
		t.Error("expected error for missing session_key, got nil")
	}

	// Empty session_key
	err = p.Login(ctx, map[string]string{"session_key": ""})
	if err == nil {
		t.Error("expected error for empty session_key, got nil")
	}

	// Valid session_key
	validKey := "session_cookie_value"
	err = p.Login(ctx, map[string]string{"session_key": validKey})
	if err != nil {
		t.Errorf("expected no error for valid session_key, got %v", err)
	}
	if p.sessionKey != validKey {
		t.Errorf("expected sessionKey to be '%s', got '%s'", validKey, p.sessionKey)
	}
}

func TestClaudeProvider_FetchUsageInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptySessionKey", func(t *testing.T) {
		p := NewClaudeProvider()
		_, err := p.FetchUsageInfo(ctx)
		if !errors.Is(err, provider.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized for empty session key, got %v", err)
		}
	})

	t.Run("Unauthorized_Organizations", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "invalid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if !errors.Is(err, provider.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("UnexpectedStatus_Organizations", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if err == nil || err.Error() != "unexpected status fetching organizations: 500" {
			t.Errorf("expected unexpected status error, got %v", err)
		}
	})

	t.Run("EmptyOrgs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if err == nil || err.Error() != "no organizations found" {
			t.Errorf("expected no organizations found error, got %v", err)
		}
	})

	t.Run("Unauthorized_BillingInfo", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "org-123"}})
				return
			}
			if r.URL.Path == "/organizations/org-123/billing_info" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "invalid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if !errors.Is(err, provider.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("UnexpectedStatus_BillingInfo", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "org-123"}})
				return
			}
			if r.URL.Path == "/organizations/org-123/billing_info" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if err == nil || err.Error() != "unexpected status fetching billing info: 500" {
			t.Errorf("expected unexpected status error, got %v", err)
		}
	})

	t.Run("Success_RFC3339", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				cookie := r.Header.Get("Cookie")
				if cookie != "sessionKey=valid_key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "org-123"}})
				return
			}
			if r.URL.Path == "/organizations/org-123/billing_info" {
				w.WriteHeader(http.StatusOK)
				billingResp := `{"billing_period": {"end_date": "2024-05-01T00:00:00Z"}}`
				w.Write([]byte(billingResp))
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		info, err := p.FetchUsageInfo(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedDate, _ := time.Parse(time.RFC3339, "2024-05-01T00:00:00Z")
		if !info.ResetDate.Equal(expectedDate) {
			t.Errorf("expected reset date %v, got %v", expectedDate, info.ResetDate)
		}
	})

	t.Run("Success_DateOnly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "org-123"}})
				return
			}
			if r.URL.Path == "/organizations/org-123/billing_info" {
				w.WriteHeader(http.StatusOK)
				billingResp := `{"billing_period": {"end_date": "2024-05-01"}}`
				w.Write([]byte(billingResp))
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		info, err := p.FetchUsageInfo(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedDate, _ := time.Parse("2006-01-02", "2024-05-01")
		if !info.ResetDate.Equal(expectedDate) {
			t.Errorf("expected reset date %v, got %v", expectedDate, info.ResetDate)
		}
	})

	t.Run("NoEndDate", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/organizations" {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "org-123"}})
				return
			}
			if r.URL.Path == "/organizations/org-123/billing_info" {
				w.WriteHeader(http.StatusOK)
				billingResp := `{"billing_period": {}}`
				w.Write([]byte(billingResp))
				return
			}
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}))
		defer server.Close()

		p := NewClaudeProvider()
		p.baseURL = server.URL
		_ = p.Login(ctx, map[string]string{"session_key": "valid_key"})

		_, err := p.FetchUsageInfo(ctx)
		if err == nil || err.Error() != "no end_date found in billing info" {
			t.Errorf("expected no end_date error, got %v", err)
		}
	})
}
