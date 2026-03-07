package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type contextKey string

const UserCtxKey contextKey = "user"

func UserFromContext(ctx context.Context) (*TelegramUser, bool) {
	u, ok := ctx.Value(UserCtxKey).(*TelegramUser)
	return u, ok
}

// ValidateTelegramLogin verifies the data received from Telegram Login Widget.
// Returns true, the user object, and nil if valid.
func ValidateTelegramLogin(botToken string, data url.Values) (bool, *TelegramUser, error) {
	hash := data.Get("hash")
	if hash == "" {
		return false, nil, fmt.Errorf("missing hash")
	}

	var dataCheckArr []string
	for k, v := range data {
		if k == "hash" {
			continue
		}
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, v[0]))
	}

	sort.Strings(dataCheckArr)
	dataCheckString := strings.Join(dataCheckArr, "\n")

	secretKeyHash := sha256.New()
	secretKeyHash.Write([]byte(botToken))
	secretKey := secretKeyHash.Sum(nil)

	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if calculatedHash != hash {
		return false, nil, fmt.Errorf("invalid hash")
	}

	authDateStr := data.Get("auth_date")
	authDateInt, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return false, nil, fmt.Errorf("invalid auth_date")
	}

	authDate := time.Unix(authDateInt, 0)
	if time.Since(authDate) > 24*time.Hour { // e.g., expire after 24h
		return false, nil, fmt.Errorf("auth_date is too old")
	}

	idStr := data.Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return false, nil, fmt.Errorf("invalid user id")
	}

	user := &TelegramUser{
		ID:        id,
		FirstName: data.Get("first_name"),
		LastName:  data.Get("last_name"),
		Username:  data.Get("username"),
	}

	return true, user, nil
}

// CreateSessionToken creates an HMAC-signed token for the given user ID.
func CreateSessionToken(userID int64, secret string) string {
	payload := fmt.Sprintf("%d:%d", userID, time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s:%s", payload, signature)
}

// VerifySessionToken verifies the HMAC-signed token and extracts the user ID.
func VerifySessionToken(token, secret string) (int64, bool) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return 0, false
	}

	userIDStr := parts[0]
	timestampStr := parts[1]
	signature := parts[2]

	payload := fmt.Sprintf("%s:%s", userIDStr, timestampStr)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return 0, false
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, false
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0, false
	}

	// Token expires after 30 days
	if time.Since(time.Unix(timestamp, 0)) > 30*24*time.Hour {
		return 0, false
	}

	return userID, true
}

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_session")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			userID, ok := VerifySessionToken(cookie.Value, secret)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid session"})
				return
			}

			user := &TelegramUser{ID: userID}
			ctx := context.WithValue(r.Context(), UserCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FetchTelegramBotUsername queries the Telegram API to get the bot's username using its token.
func FetchTelegramBotUsername(token string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch bot info: status %d", resp.StatusCode)
	}

	var data struct {
		Ok     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if !data.Ok || data.Result.Username == "" {
		return "", fmt.Errorf("failed to parse bot username from response")
	}

	return data.Result.Username, nil
}
