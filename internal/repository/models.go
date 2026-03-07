package repository

import "time"

type Subscription struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Service    string    `json:"service"`
	BillingDay int64     `json:"billing_day"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WebpushSubscription struct {
	ID        int64     `json:"id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}

type WebpushSubscriptionParams struct {
	Endpoint string
	P256dh   string
	Auth     string
}

type TelegramChat struct {
	ID        int64     `json:"id"`
	ChatID    string    `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationLog struct {
	ID             int64     `json:"id"`
	SubscriptionID int64     `json:"subscription_id"`
	Channel        string    `json:"channel"`
	Message        string    `json:"message"`
	SentAt         time.Time `json:"sent_at"`
}

type CreateSubscriptionParams struct {
	Name       string
	Service    string
	BillingDay int64
	Notes      string
}

type UpdateSubscriptionParams struct {
	ID         int64
	Name       string
	Service    string
	BillingDay int64
	Notes      string
}

type CreateNotificationLogParams struct {
	SubscriptionID int64
	Channel        string
	Message        string
}
