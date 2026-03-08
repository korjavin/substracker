package auth

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
)

func TestSessionTokenRoundtrip(t *testing.T) {
	secret := "test-secret"
	userID := int64(12345)

	token := CreateSessionToken(userID, secret)

	extractedID, ok := VerifySessionToken(token, secret)
	if !ok {
		t.Fatalf("expected token to verify successfully")
	}

	if extractedID != userID {
		t.Errorf("expected user ID %d, got %d", userID, extractedID)
	}
}

func TestVerifySessionToken_InvalidToken(t *testing.T) {
	secret := "test-secret"
	token := "invalid:token"

	_, ok := VerifySessionToken(token, secret)
	if ok {
		t.Errorf("expected token verification to fail for malformed token")
	}

	userID := int64(12345)
	validToken := CreateSessionToken(userID, secret)

	parts := strings.Split(validToken, ":")
	parts[2] = "tampered"
	tamperedToken := strings.Join(parts, ":")

	_, ok = VerifySessionToken(tamperedToken, secret)
	if ok {
		t.Errorf("expected token verification to fail for tampered signature")
	}

	_, ok = VerifySessionToken(validToken, "wrong-secret")
	if ok {
		t.Errorf("expected token verification to fail with wrong secret")
	}
}

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	middleware := AuthMiddleware(secret)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Errorf("expected user in context")
		}
		if user.ID != 12345 {
			t.Errorf("expected user ID 12345, got %d", user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("ValidSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		token := CreateSessionToken(12345, secret)
		req.AddCookie(&http.Cookie{Name: "auth_session", Value: token})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("NoSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("InvalidSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "auth_session", Value: "invalid"})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

func TestValidateTelegramLogin(t *testing.T) {
	botToken := "test-bot-token"

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

		h := hmac.New(sha256.New, secretKey)
		h.Write([]byte(dataCheckString))
		hash := hex.EncodeToString(h.Sum(nil))

		data.Set("hash", hash)
		return data
	}

	t.Run("ValidLogin", func(t *testing.T) {
		now := time.Now().Unix()
		data := generateValidData(111, "testuser", now)

		ok, user, err := ValidateTelegramLogin(botToken, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Errorf("expected validation to succeed")
		}
		if user.ID != 111 {
			t.Errorf("expected user ID 111, got %d", user.ID)
		}
		if user.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", user.Username)
		}
	})

	t.Run("InvalidHash", func(t *testing.T) {
		now := time.Now().Unix()
		data := generateValidData(111, "testuser", now)
		data.Set("hash", "invalidhash")

		ok, _, err := ValidateTelegramLogin(botToken, data)
		if ok {
			t.Errorf("expected validation to fail")
		}
		if err == nil || !strings.Contains(err.Error(), "invalid hash") {
			t.Errorf("expected invalid hash error, got %v", err)
		}
	})

	t.Run("MissingHash", func(t *testing.T) {
		now := time.Now().Unix()
		data := generateValidData(111, "testuser", now)
		data.Del("hash")

		ok, _, err := ValidateTelegramLogin(botToken, data)
		if ok {
			t.Errorf("expected validation to fail")
		}
		if err == nil || !strings.Contains(err.Error(), "missing hash") {
			t.Errorf("expected missing hash error, got %v", err)
		}
	})

	t.Run("ExpiredAuthDate", func(t *testing.T) {
		past := time.Now().Add(-25 * time.Hour).Unix()
		data := generateValidData(111, "testuser", past)

		ok, _, err := ValidateTelegramLogin(botToken, data)
		if ok {
			t.Errorf("expected validation to fail")
		}
		if err == nil || !strings.Contains(err.Error(), "auth_date is too old") {
			t.Errorf("expected old auth_date error, got %v", err)
		}
	})
}
