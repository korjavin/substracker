package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	webpush "github.com/SherClockHolmes/webpush-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/korjavin/substracker/internal/repository"
)

type NotificationConfig struct {
	TelegramBotToken string
	VAPIDPublicKey   string
	VAPIDPrivateKey  string
	VAPIDSubject     string
}

type NotificationService struct {
	repo  *repository.Queries
	cfg   NotificationConfig
	tgBot *tgbotapi.BotAPI
}

func NewNotificationService(repo *repository.Queries, cfg NotificationConfig) *NotificationService {
	svc := &NotificationService{repo: repo, cfg: cfg}

	if cfg.TelegramBotToken != "" {
		bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
		if err != nil {
			slog.Error("failed to create telegram bot", "error", err)
		} else {
			svc.tgBot = bot
			slog.Info("telegram bot initialized", "username", bot.Self.UserName)
		}
	}

	return svc
}

// SendAll sends message via all configured channels.
// subID > 0 means it's tied to a real subscription and will be logged.
func (s *NotificationService) SendAll(ctx context.Context, userID, subID int64, message string) {
	s.sendWebPush(ctx, userID, subID, message)
	s.sendTelegram(ctx, userID, subID, message)
}

func (s *NotificationService) sendWebPush(ctx context.Context, userID, subID int64, message string) {
	if s.cfg.VAPIDPublicKey == "" || s.cfg.VAPIDPrivateKey == "" {
		return
	}

	subs, err := s.repo.ListWebPushSubscriptions(ctx, userID)
	if err != nil {
		slog.Error("list webpush subscriptions", "error", err)
		return
	}

	payload, _ := json.Marshal(map[string]string{"message": message})

	for _, sub := range subs {
		pushSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}
		resp, err := webpush.SendNotification(payload, pushSub, &webpush.Options{
			VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
			VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
			Subscriber:      s.cfg.VAPIDSubject,
			TTL:             30,
		})
		if err != nil {
			slog.Error("send webpush notification", "endpoint", sub.Endpoint, "error", err)
			continue
		}
		resp.Body.Close()

		if subID > 0 {
			_ = s.repo.CreateNotificationLog(ctx, repository.CreateNotificationLogParams{
				SubscriptionID: subID,
				Channel:        "webpush",
				Message:        message,
			})
		}
	}
}

func (s *NotificationService) sendTelegram(ctx context.Context, userID, subID int64, message string) {
	if s.tgBot == nil {
		return
	}

	chats, err := s.repo.ListTelegramChats(ctx, userID)
	if err != nil {
		slog.Error("list telegram chats", "error", err)
		return
	}

	for _, chat := range chats {
		chatIDInt, err := strconv.ParseInt(chat.ChatID, 10, 64)
		if err != nil {
			slog.Error("invalid telegram chat_id", "chat_id", chat.ChatID, "error", err)
			continue
		}

		msg := tgbotapi.NewMessage(chatIDInt, message)
		if _, err := s.tgBot.Send(msg); err != nil {
			slog.Error("send telegram message", "chat_id", chat.ChatID, "error", err)
			continue
		}

		if subID > 0 {
			_ = s.repo.CreateNotificationLog(ctx, repository.CreateNotificationLogParams{
				SubscriptionID: subID,
				Channel:        "telegram",
				Message:        message,
			})
		}
	}
}
