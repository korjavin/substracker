package openaiprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/korjavin/substracker/internal/provider"
)

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider()
	if p.Name() != "OpenAI" {
		t.Errorf("expected OpenAI, got %s", p.Name())
	}
}

func TestOpenAIProvider_Login(t *testing.T) {
	p := NewOpenAIProvider()

	err := p.Login(context.Background(), map[string]string{})
	if err == nil {
		t.Error("expected error for empty credentials, got nil")
	}

	err = p.Login(context.Background(), map[string]string{"session_token": ""})
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}

	err = p.Login(context.Background(), map[string]string{"session_token": "token123"})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	p.mu.RLock()
	token := p.sessionToken
	p.mu.RUnlock()

	if token != "token123" {
		t.Errorf("expected token123, got %s", token)
	}
}

func TestOpenAIProvider_FetchUsageInfo(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		handler      http.HandlerFunc
		wantError    bool
		wantUnauthorized bool
		wantTime     bool
	}{
		{
			name:      "unauthorized no token",
			token:     "",
			handler:   func(w http.ResponseWriter, r *http.Request) {},
			wantError: true,
			wantUnauthorized: true,
		},
		{
			name:  "server returns 401",
			token: "badtoken",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantError:        true,
			wantUnauthorized: true,
		},
		{
			name:  "server returns 500",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantError: true,
		},
		{
			name:  "server returns malformed json",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{malformed json}`))
			},
			wantError: true,
		},
		{
			name:  "server returns ok with access_until",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				cookieHeader := r.Header.Get("Cookie")
				if cookieHeader != "__Secure-next-auth.session-token=token123" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"object": "billing_subscription", "access_until": 1715356800}`))
			},
			wantError: false,
			wantTime:  true,
		},
		{
			name:  "server returns ok without access_until fallback",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"object": "billing_subscription"}`))
			},
			wantError: false,
			wantTime:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			p := NewOpenAIProvider()
			p.baseURL = server.URL
			if tt.token != "" {
				p.Login(context.Background(), map[string]string{"session_token": tt.token})
			}

			info, err := p.FetchUsageInfo(context.Background())

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.wantUnauthorized && err != provider.ErrUnauthorized && err.Error() != "openaiprovider: unauthorized: relogin required" {
					t.Errorf("expected unauthorized error, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantTime && info.ResetDate.IsZero() {
					t.Errorf("expected valid reset date, got zero")
				}
			}
		})
	}
}