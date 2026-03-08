package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/provider/claudeprovider"
	"github.com/korjavin/substracker/internal/provider/googleoneprovider"
	"github.com/korjavin/substracker/internal/repository"
	"github.com/korjavin/substracker/internal/service"
)

type Handler struct {
	repo              *repository.Queries
	notifSvc          *service.NotificationService
	vapidPublicKey    string
	claudeProvider    provider.Provider
	googleOneProvider provider.Provider
}

func NewHandler(repo *repository.Queries, notifSvc *service.NotificationService, vapidPublicKey string) *Handler {
	return &Handler{
		repo:              repo,
		notifSvc:          notifSvc,
		vapidPublicKey:    vapidPublicKey,
		claudeProvider:    claudeprovider.NewClaudeProvider(),
		googleOneProvider: googleoneprovider.NewGoogleOneProvider(),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	// Claude Provider
	mux.HandleFunc("GET /api/providers/claude/login-info", h.claudeLoginInfo)
	mux.HandleFunc("POST /api/providers/claude/login", h.claudeLogin)
	mux.HandleFunc("GET /api/providers/claude/usage", h.claudeUsage)

	// Google One Provider
	mux.HandleFunc("GET /api/providers/googleone/login-info", h.googleOneLoginInfo)
	mux.HandleFunc("POST /api/providers/googleone/login", h.googleOneLogin)
	mux.HandleFunc("GET /api/providers/googleone/usage", h.googleOneUsage)

	// Subscriptions
	mux.HandleFunc("GET /api/subscriptions", h.listSubscriptions)
	mux.HandleFunc("POST /api/subscriptions", h.createSubscription)
	mux.HandleFunc("GET /api/subscriptions/{id}", h.getSubscription)
	mux.HandleFunc("PUT /api/subscriptions/{id}", h.updateSubscription)
	mux.HandleFunc("DELETE /api/subscriptions/{id}", h.deleteSubscription)

	// Web Push
	mux.HandleFunc("GET /api/vapid-public-key", h.getVapidPublicKey)
	mux.HandleFunc("POST /api/webpush/subscribe", h.webpushSubscribe)
	mux.HandleFunc("DELETE /api/webpush/subscribe", h.webpushUnsubscribe)
	mux.HandleFunc("GET /api/webpush/subscriptions", h.listWebPushSubs)

	// Telegram
	mux.HandleFunc("GET /api/telegram/chats", h.listTelegramChats)
	mux.HandleFunc("POST /api/telegram/chats", h.addTelegramChat)
	mux.HandleFunc("DELETE /api/telegram/chats/{chatId}", h.deleteTelegramChat)

	// Notifications
	mux.HandleFunc("GET /api/notifications/log", h.listNotificationLogs)
	mux.HandleFunc("POST /api/notifications/test", h.testNotification)

	// Static files (must come last)
	mux.Handle("/", http.FileServer(http.Dir("web")))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Health ---

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Claude Provider ---

func (h *Handler) claudeLoginInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"url":          "https://claude.ai/",
		"instructions": "Log in to claude.ai, open Developer Tools -> Application -> Cookies, and copy the value of the 'sessionKey' cookie.",
	})
}

func (h *Handler) claudeLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionKey string `json:"session_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.claudeProvider.Login(r.Context(), map[string]string{"session_key": req.SessionKey})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_in"})
}

func (h *Handler) claudeUsage(w http.ResponseWriter, r *http.Request) {
	info, err := h.claudeProvider.FetchUsageInfo(r.Context())
	if err != nil {
		if errors.Is(err, provider.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "relogin_required"})
			return
		}
		slog.Error("fetch claude usage", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch usage")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// --- Google One Provider ---

func (h *Handler) googleOneLoginInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"url":          "https://one.google.com/",
		"instructions": "Log in to one.google.com, open Developer Tools -> Application -> Cookies, and copy the value of the 'SID' cookie.",
	})
}

func (h *Handler) googleOneLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionCookie string `json:"session_cookie"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.googleOneProvider.Login(r.Context(), map[string]string{"session_cookie": req.SessionCookie})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_in"})
}

func (h *Handler) googleOneUsage(w http.ResponseWriter, r *http.Request) {
	info, err := h.googleOneProvider.FetchUsageInfo(r.Context())
	if err != nil {
		if errors.Is(err, provider.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "relogin_required"})
			return
		}
		slog.Error("fetch google one usage", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch usage")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// --- Subscriptions ---

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := h.repo.ListSubscriptions(r.Context())
	if err != nil {
		slog.Error("list subscriptions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	if subs == nil {
		subs = []repository.Subscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Service    string `json:"service"`
		BillingDay int64  `json:"billing_day"`
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Service == "" || req.BillingDay < 1 || req.BillingDay > 31 {
		writeError(w, http.StatusBadRequest, "name, service, and billing_day (1-31) are required")
		return
	}

	sub, err := h.repo.CreateSubscription(r.Context(), repository.CreateSubscriptionParams{
		Name:       req.Name,
		Service:    req.Service,
		BillingDay: req.BillingDay,
		Notes:      req.Notes,
	})
	if err != nil {
		slog.Error("create subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sub, err := h.repo.GetSubscription(r.Context(), id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if err != nil {
		slog.Error("get subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Name       string `json:"name"`
		Service    string `json:"service"`
		BillingDay int64  `json:"billing_day"`
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Service == "" || req.BillingDay < 1 || req.BillingDay > 31 {
		writeError(w, http.StatusBadRequest, "name, service, and billing_day (1-31) are required")
		return
	}

	sub, err := h.repo.UpdateSubscription(r.Context(), repository.UpdateSubscriptionParams{
		ID:         id,
		Name:       req.Name,
		Service:    req.Service,
		BillingDay: req.BillingDay,
		Notes:      req.Notes,
	})
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if err != nil {
		slog.Error("update subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update subscription")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.DeleteSubscription(r.Context(), id); err != nil {
		slog.Error("delete subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Web Push ---

func (h *Handler) getVapidPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"key": h.vapidPublicKey})
}

func (h *Handler) webpushSubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Endpoint == "" {
		writeError(w, http.StatusBadRequest, "endpoint is required")
		return
	}

	if err := h.repo.UpsertWebPushSubscription(r.Context(), repository.WebpushSubscriptionParams{
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	}); err != nil {
		slog.Error("upsert webpush subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save subscription")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "subscribed"})
}

func (h *Handler) webpushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.repo.DeleteWebPushSubscription(r.Context(), req.Endpoint); err != nil {
		slog.Error("delete webpush subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listWebPushSubs(w http.ResponseWriter, r *http.Request) {
	subs, err := h.repo.ListWebPushSubscriptions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	if subs == nil {
		subs = []repository.WebpushSubscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

// --- Telegram ---

func (h *Handler) listTelegramChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.repo.ListTelegramChats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chats")
		return
	}
	if chats == nil {
		chats = []repository.TelegramChat{}
	}
	writeJSON(w, http.StatusOK, chats)
}

func (h *Handler) addTelegramChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ChatID == "" {
		writeError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if err := h.repo.CreateTelegramChat(r.Context(), req.ChatID); err != nil {
		slog.Error("create telegram chat", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to add chat")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (h *Handler) deleteTelegramChat(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("chatId")
	if err := h.repo.DeleteTelegramChat(r.Context(), chatID); err != nil {
		slog.Error("delete telegram chat", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove chat")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Notification Log ---

func (h *Handler) listNotificationLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.repo.ListNotificationLogs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list logs")
		return
	}
	if logs == nil {
		logs = []repository.NotificationLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *Handler) testNotification(w http.ResponseWriter, r *http.Request) {
	h.notifSvc.SendAll(context.Background(), 0, "Test notification from SubsTracker! If you see this, notifications are working.")
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
