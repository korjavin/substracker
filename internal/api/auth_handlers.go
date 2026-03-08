package api

import (
	"log/slog"
	"net/http"

	"github.com/korjavin/substracker/internal/auth"
)

func (h *Handler) telegramCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid form data"))
		return
	}

	ok, user, err := auth.ValidateTelegramLogin(h.telegramBotToken, r.Form)
	if err != nil || !ok {
		slog.Error("telegram login validation failed", "error", err)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("unauthorized"))
		return
	}

	sessionToken := auth.CreateSessionToken(user.ID, h.sessionSecret)

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_session",
		Value:    sessionToken,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_session",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *Handler) authStatus(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_session")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized", "bot_username": h.telegramBotUsername})
		return
	}

	userID, ok := auth.VerifySessionToken(cookie.Value, h.sessionSecret)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized", "bot_username": h.telegramBotUsername})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           userID,
		"bot_username": h.telegramBotUsername,
	})
}
