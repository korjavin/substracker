package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/korjavin/substracker/internal/auth"
	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/provider/claudeprovider"
	"github.com/korjavin/substracker/internal/provider/googleoneprovider"
	"github.com/korjavin/substracker/internal/provider/openaiprovider"
	"github.com/korjavin/substracker/internal/provider/zaiprovider"
	"github.com/korjavin/substracker/internal/repository"
	"github.com/korjavin/substracker/internal/service"
)

type Handler struct {
	repo                *repository.Queries
	notifSvc            *service.NotificationService
	vapidPublicKey      string
	sessionSecret       string
	telegramBotToken    string
	telegramBotUsername string
	claudeProvider      provider.Provider
	googleOneProvider   provider.Provider
	zaiProvider         provider.Provider
	openaiProvider      provider.Provider
}

func NewHandler(repo *repository.Queries, notifSvc *service.NotificationService, vapidPublicKey, sessionSecret, telegramBotToken, telegramBotUsername string) *Handler {
	return &Handler{
		repo:                repo,
		notifSvc:            notifSvc,
		vapidPublicKey:      vapidPublicKey,
		sessionSecret:       sessionSecret,
		telegramBotToken:    telegramBotToken,
		telegramBotUsername: telegramBotUsername,
		claudeProvider:      claudeprovider.NewClaudeProvider(),
		googleOneProvider:   googleoneprovider.NewGoogleOneProvider(),
		zaiProvider:         zaiprovider.NewZAIProvider(),
		openaiProvider:      openaiprovider.NewOpenAIProvider(),
	}
}

// GetZAIProvider returns the Z.ai provider instance.
func (h *Handler) GetZAIProvider() provider.Provider {
	return h.zaiProvider
}

// GetClaudeProvider returns the Claude provider instance.
func (h *Handler) GetClaudeProvider() provider.Provider {
	return h.claudeProvider
}

// GetGoogleOneProvider returns the Google One provider instance.
func (h *Handler) GetGoogleOneProvider() provider.Provider {
	return h.googleOneProvider
}

// GetOpenAIProvider returns the OpenAI provider instance.
func (h *Handler) GetOpenAIProvider() provider.Provider {
	return h.openaiProvider
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	// Auth Check
	mux.HandleFunc("GET /api/auth/me", h.authStatus)

	apiMux := http.NewServeMux()

	// Subscriptions
	apiMux.HandleFunc("GET /api/subscriptions", h.listSubscriptions)
	apiMux.HandleFunc("POST /api/subscriptions", h.createSubscription)
	apiMux.HandleFunc("GET /api/subscriptions/{id}", h.getSubscription)
	apiMux.HandleFunc("PUT /api/subscriptions/{id}", h.updateSubscription)
	apiMux.HandleFunc("DELETE /api/subscriptions/{id}", h.deleteSubscription)
	apiMux.HandleFunc("GET /api/subscriptions/usage/cached", h.cachedSubscriptionUsage)
	apiMux.HandleFunc("POST /api/subscriptions/usage/refresh", h.refreshSubscriptionsUsage)

	// Web Push
	apiMux.HandleFunc("GET /api/vapid-public-key", h.getVapidPublicKey)
	apiMux.HandleFunc("POST /api/webpush/subscribe", h.webpushSubscribe)
	apiMux.HandleFunc("DELETE /api/webpush/subscribe", h.webpushUnsubscribe)
	apiMux.HandleFunc("GET /api/webpush/subscriptions", h.listWebPushSubs)

	// Telegram
	apiMux.HandleFunc("GET /api/telegram/chats", h.listTelegramChats)
	apiMux.HandleFunc("POST /api/telegram/chats", h.addTelegramChat)
	apiMux.HandleFunc("DELETE /api/telegram/chats/{chatId}", h.deleteTelegramChat)

	// Notifications
	apiMux.HandleFunc("GET /api/notifications/log", h.listNotificationLogs)
	apiMux.HandleFunc("POST /api/notifications/test", h.testNotification)

	mux.Handle("/api/", auth.AuthMiddleware(h.sessionSecret)(apiMux))

	// Auth routes
	mux.HandleFunc("GET /auth/telegram/callback", h.telegramCallback)
	mux.HandleFunc("GET /auth/logout", h.logout)

	// Protect root explicitly, let /login through unauthenticated
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/login.html")
	})

	fs := http.FileServer(http.Dir("web"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only check auth for exact matches of / or /index.html
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			cookie, err := r.Cookie("auth_session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			if _, ok := auth.VerifySessionToken(cookie.Value, h.sessionSecret); !ok {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
		}
		fs.ServeHTTP(w, r)
	})
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

// --- Subscriptions ---

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subs, err := h.repo.ListSubscriptions(r.Context(), user.ID)
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name       string `json:"name"`
		Service    string `json:"service"`
		BillingDay int64  `json:"billing_day"`
		Notes      string `json:"notes"`
		AuthToken  string `json:"auth_token"`
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
		UserID:     user.ID,
		Name:       req.Name,
		Service:    req.Service,
		BillingDay: req.BillingDay,
		Notes:      req.Notes,
		AuthToken:  req.AuthToken,
	})
	if err != nil {
		slog.Error("create subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sub, err := h.repo.GetSubscription(r.Context(), id, user.ID)
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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
		AuthToken  string `json:"auth_token"`
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
		UserID:     user.ID,
		Name:       req.Name,
		Service:    req.Service,
		BillingDay: req.BillingDay,
		Notes:      req.Notes,
		AuthToken:  req.AuthToken,
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.DeleteSubscription(r.Context(), id, user.ID); err != nil {
		slog.Error("delete subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cachedSubscriptionUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	usages, err := h.repo.ListSubscriptionUsageByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list cached sub usage", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get cached sub usage")
		return
	}
	if usages == nil {
		usages = []repository.SubscriptionUsage{}
	}

	writeJSON(w, http.StatusOK, usages)
}

func (h *Handler) refreshSubscriptionsUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subs, err := h.repo.ListSubscriptions(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}

	hasReloginError := false

	for _, sub := range subs {
		if sub.AuthToken == "" {
			continue
		}

		var p provider.Provider
		switch sub.Service {
		case "claude":
			p = h.claudeProvider
		case "openai":
			p = h.openaiProvider
		case "googleone":
			p = h.googleOneProvider
		case "zai":
			p = h.zaiProvider
		default:
			continue
		}

		creds := map[string]string{
			"session_key":    sub.AuthToken,
			"session_token":  sub.AuthToken,
			"session_cookie": sub.AuthToken,
		}

		info, err := p.FetchUsageInfo(r.Context(), creds)
		if err != nil {
			slog.Error("failed to fetch usage info for sub", "subID", sub.ID, "error", err)
			if errors.Is(err, provider.ErrUnauthorized) {
				hasReloginError = true
			}
			continue
		}

		_ = h.repo.UpsertSubscriptionUsage(r.Context(), repository.UpsertSubscriptionUsageParams{
			SubscriptionID:      sub.ID,
			CurrentUsageSeconds: info.CurrentUsageSeconds,
			TotalLimitSeconds:   info.TotalLimitSeconds,
			IsBlocked:           info.IsBlocked,
		})
	}

	if hasReloginError {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "relogin_required", "message": "Failed to refresh some subscriptions due to invalid auth tokens. Please check your tokens."})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}

// --- Web Push ---

func (h *Handler) getVapidPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"key": h.vapidPublicKey})
}

func (h *Handler) webpushSubscribe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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
		UserID:   user.ID,
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.repo.DeleteWebPushSubscription(r.Context(), req.Endpoint, user.ID); err != nil {
		slog.Error("delete webpush subscription", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listWebPushSubs(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subs, err := h.repo.ListWebPushSubscriptions(r.Context(), user.ID)
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chats, err := h.repo.ListTelegramChats(r.Context(), user.ID)
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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
	if err := h.repo.CreateTelegramChat(r.Context(), req.ChatID, user.ID); err != nil {
		slog.Error("create telegram chat", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to add chat")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (h *Handler) deleteTelegramChat(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := r.PathValue("chatId")
	if err := h.repo.DeleteTelegramChat(r.Context(), chatID, user.ID); err != nil {
		slog.Error("delete telegram chat", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove chat")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Notification Log ---

func (h *Handler) listNotificationLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	logs, err := h.repo.ListNotificationLogs(r.Context(), user.ID)
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	h.notifSvc.SendAll(context.Background(), user.ID, 0, "Test notification from SubsTracker! If you see this, notifications are working.")
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
