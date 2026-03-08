package googleoneprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/provider"
)

func TestGoogleOneProvider_Name(t *testing.T) {
	p := NewGoogleOneProvider()
	if name := p.Name(); name != "Google One" {
		t.Errorf("expected Name() to return 'Google One', got '%s'", name)
	}
}

func TestGoogleOneProvider_Login(t *testing.T) {
	p := NewGoogleOneProvider()

	// Test successful login
	err := p.Login(context.Background(), map[string]string{"session_cookie": "test_cookie_123"})
	if err != nil {
		t.Errorf("unexpected error on valid login: %v", err)
	}

	p.mu.RLock()
	if p.sessionCookie != "test_cookie_123" {
		t.Errorf("expected sessionCookie to be 'test_cookie_123', got '%s'", p.sessionCookie)
	}
	p.mu.RUnlock()

	// Test missing credential
	err = p.Login(context.Background(), map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "missing or empty session_cookie") {
		t.Errorf("expected missing session_cookie error, got %v", err)
	}

	// Test empty credential
	err = p.Login(context.Background(), map[string]string{"session_cookie": ""})
	if err == nil || !strings.Contains(err.Error(), "missing or empty session_cookie") {
		t.Errorf("expected empty session_cookie error, got %v", err)
	}
}

func TestGoogleOneProvider_FetchUsageInfo(t *testing.T) {
	t.Run("NoSessionCookie", func(t *testing.T) {
		p := NewGoogleOneProvider()
		_, err := p.FetchUsageInfo(context.Background())
		if err == nil || !errors.Is(err, provider.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized when no cookie is set, got %v", err)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		p := NewGoogleOneProvider()
		p.baseURL = ts.URL
		p.sessionCookie = "invalid_cookie"

		_, err := p.FetchUsageInfo(context.Background())
		if err == nil || !errors.Is(err, provider.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized on 401 response, got %v", err)
		}
	})

	t.Run("UnexpectedStatus", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		p := NewGoogleOneProvider()
		p.baseURL = ts.URL
		p.sessionCookie = "valid_cookie"

		_, err := p.FetchUsageInfo(context.Background())
		if err == nil || !strings.Contains(err.Error(), "unexpected status") {
			t.Errorf("expected unexpected status error on 500 response, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie := r.Header.Get("Cookie"); cookie != "SID=valid_cookie" {
				t.Errorf("expected Cookie: SID=valid_cookie, got %s", cookie)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"billing_period": {"end_date": "2026-04-01T00:00:00Z"}}`))
		}))
		defer ts.Close()

		p := NewGoogleOneProvider()
		p.baseURL = ts.URL
		p.sessionCookie = "valid_cookie"

		info, err := p.FetchUsageInfo(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("expected usage info, got nil")
		}

		expectedDate, _ := time.Parse(time.RFC3339, "2026-04-01T00:00:00Z")
		if !info.ResetDate.Equal(expectedDate) {
			t.Errorf("expected ResetDate %v, got %v", expectedDate, info.ResetDate)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"invalid_json`))
		}))
		defer ts.Close()

		p := NewGoogleOneProvider()
		p.baseURL = ts.URL
		p.sessionCookie = "valid_cookie"

		_, err := p.FetchUsageInfo(context.Background())
		if err == nil || !strings.Contains(err.Error(), "failed to decode subscriptions") {
			t.Errorf("expected JSON decode error, got %v", err)
		}
	})

	t.Run("MissingEndDate", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"billing_period": {}}`))
		}))
		defer ts.Close()

		p := NewGoogleOneProvider()
		p.baseURL = ts.URL
		p.sessionCookie = "valid_cookie"

		_, err := p.FetchUsageInfo(context.Background())
		if err == nil || !strings.Contains(err.Error(), "no end_date found") {
			t.Errorf("expected no end_date found error, got %v", err)
		}
	})
}
