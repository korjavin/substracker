package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/korjavin/substracker/internal/api"
	"github.com/korjavin/substracker/internal/db"
	"github.com/korjavin/substracker/internal/middleware"
	"github.com/korjavin/substracker/internal/repository"
	"github.com/korjavin/substracker/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repo := repository.New(database)

	notifCfg := service.NotificationConfig{
		TelegramBotToken: os.Getenv("TG_BOT_TOKEN"),
		VAPIDPublicKey:   os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:  os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:     os.Getenv("VAPID_SUBJECT"),
	}
	notifSvc := service.NewNotificationService(repo, notifCfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := service.NewScheduler(repo, notifSvc, logger)
	go scheduler.Run(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5454"
	}

	mux := http.NewServeMux()
	limiter := middleware.NewRateLimiter(10, 20) // 10 req/s, burst 20

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		slog.Error("SESSION_SECRET environment variable is required")
		os.Exit(1)
	}
	telegramBotUsername := os.Getenv("TELEGRAM_BOT_USERNAME")

	handler := api.NewHandler(repo, notifSvc, notifCfg.VAPIDPublicKey, sessionSecret, notifCfg.TelegramBotToken, telegramBotUsername)
	handler.Register(mux)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      middleware.Logging(limiter.Limit(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("server starting", "port", port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("stopped")
}
