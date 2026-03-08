package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/korjavin/substracker/internal/auth"
)

func TestAuthStatus(t *testing.T) {
	sessionSecret := "test-secret"
	botUsername := "test_bot"

	h := &Handler{
		sessionSecret:       sessionSecret,
		telegramBotUsername: botUsername,
	}

	t.Run("ValidSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		token := auth.CreateSessionToken(12345, sessionSecret)
		req.AddCookie(&http.Cookie{Name: "auth_session", Value: token})

		rr := httptest.NewRecorder()
		h.authStatus(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), "12345") {
			t.Errorf("response body should contain user id")
		}
	})

	t.Run("NoSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		rr := httptest.NewRecorder()
		h.authStatus(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("InvalidSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "auth_session", Value: "invalid:token"})
		rr := httptest.NewRecorder()
		h.authStatus(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestLogout(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.logout(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	cookieStr := rr.Header().Get("Set-Cookie")
	if !strings.Contains(cookieStr, "auth_session=") || !strings.Contains(cookieStr, "Max-Age=0") {
		t.Errorf("expected cookie to be cleared, got: %s", cookieStr)
	}
}

func TestTelegramCallback(t *testing.T) {
	botToken := "test-bot-token"
	sessionSecret := "test-secret"

	h := &Handler{
		telegramBotToken: botToken,
		sessionSecret:    sessionSecret,
	}

	generateValidData := func(id int64, username string, authDate int64) url.Values {
		data := url.Values{}
		data.Set("id", fmt.Sprintf("%d", id))
		data.Set("first_name", "Test")
		data.Set("username", username)
		data.Set("auth_date", fmt.Sprintf("%d", authDate))

		var dataCheckArr []string
		for k, v := range data {
			dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, v[0]))
		}
		sort.Strings(dataCheckArr)
		dataCheckString := strings.Join(dataCheckArr, "\n")

		secretKeyHash := sha256.New()
		secretKeyHash.Write([]byte(botToken))
		secretKey := secretKeyHash.Sum(nil)

		mac := hmac.New(sha256.New, secretKey)
		mac.Write([]byte(dataCheckString))
		hash := hex.EncodeToString(mac.Sum(nil))

		data.Set("hash", hash)
		return data
	}

	t.Run("ValidCallback", func(t *testing.T) {
		data := generateValidData(111, "testuser", time.Now().Unix())
		req := httptest.NewRequest("GET", "/auth/telegram/callback?"+data.Encode(), nil)
		rr := httptest.NewRecorder()

		h.telegramCallback(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
		}

		cookieStr := rr.Header().Get("Set-Cookie")
		if !strings.Contains(cookieStr, "auth_session=") {
			t.Errorf("expected auth_session cookie to be set")
		}
	})

	t.Run("InvalidData", func(t *testing.T) {
		data := generateValidData(111, "testuser", time.Now().Unix())
		data.Set("hash", "invalid")
		req := httptest.NewRequest("GET", "/auth/telegram/callback?"+data.Encode(), nil)
		rr := httptest.NewRecorder()

		h.telegramCallback(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})
}
