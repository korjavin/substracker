package openaiprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/korjavin/substracker/internal/provider"
)

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider()
	if p.Name() != "OpenAI" {
		t.Errorf("expected OpenAI, got %s", p.Name())
	}
}

func TestOpenAIProvider_FetchUsageInfo(t *testing.T) {
	tests := []struct {
		name             string
		token            string
		handler          http.HandlerFunc
		wantError        bool
		wantUnauthorized bool
		wantTime         bool
	}{
		{
			name:             "unauthorized no token",
			token:            "",
			handler:          func(w http.ResponseWriter, r *http.Request) {},
			wantError:        true,
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
			name:  "server returns ok with primary_window limits",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				cookieHeader := r.Header.Get("Cookie")
				authHeader := r.Header.Get("Authorization")
				if cookieHeader != "__Secure-next-auth.session-token=token123" || authHeader != "Bearer token123" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"rate_limit": {
						"primary_window": {
							"used_percent": 0.5,
							"limit_window_seconds": 3600,
							"reset_at": 1715356800
						}
					}
				}`))
			},
			wantError: false,
			wantTime:  true,
		},
		{
			name:  "server returns ok without rate_limit fallback",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"plan_type": "free"}`))
			},
			wantError: false,
			wantTime:  true,
		},
		{
			name:  "server returns unauthorized in detail body",
			token: "token123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK) // sometimes APIs return 200 with an error object
				w.Write([]byte(`{"detail": "Unauthorized"}`))
			},
			wantError: true,
			wantUnauthorized: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			p := NewOpenAIProvider()
			p.baseURL = server.URL

			creds := map[string]string{}
			if tt.token != "" {
				creds["session_token"] = tt.token
			}

			info, err := p.FetchUsageInfo(context.Background(), creds)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.wantUnauthorized && !errors.Is(err, provider.ErrUnauthorized) && !strings.Contains(err.Error(), "unauthorized") {
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
