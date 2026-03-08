package repository

import "time"

type Subscription struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Name       string    `json:"name"`
	Service    string    `json:"service"`
	BillingDay int64     `json:"billing_day"`
	Notes      string    `json:"notes"`
	AuthToken  string    `json:"auth_token"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WebpushSubscription struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}

type WebpushSubscriptionParams struct {
	UserID   int64
	Endpoint string
	P256dh   string
	Auth     string
}

type TelegramChat struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
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
	UserID     int64
	Name       string
	Service    string
	BillingDay int64
	Notes      string
	AuthToken  string
}

type UpdateSubscriptionParams struct {
	ID         int64
	UserID     int64
	Name       string
	Service    string
	BillingDay int64
	Notes      string
	AuthToken  string
}

type CreateNotificationLogParams struct {
	SubscriptionID int64
	Channel        string
	Message        string
}

type ProviderUsage struct {
	ID                  int64     `json:"id"`
	ProviderName        string    `json:"provider_name"`
	CurrentUsageSeconds int64     `json:"current_usage_seconds"`
	TotalLimitSeconds   int64     `json:"total_limit_seconds"`
	IsBlocked           bool      `json:"is_blocked"`
	FetchedAt           time.Time `json:"fetched_at"`
}

type UpsertProviderUsageParams struct {
	ProviderName        string
	CurrentUsageSeconds int64
	TotalLimitSeconds   int64
	IsBlocked           bool
}

type SubscriptionUsage struct {
	SubscriptionID      int64     `json:"subscription_id"`
	CurrentUsageSeconds int64     `json:"current_usage_seconds"`
	TotalLimitSeconds   int64     `json:"total_limit_seconds"`
	IsBlocked           bool      `json:"is_blocked"`
	FetchedAt           time.Time `json:"fetched_at"`
}

type UpsertSubscriptionUsageParams struct {
	SubscriptionID      int64
	CurrentUsageSeconds int64
	TotalLimitSeconds   int64
	IsBlocked           bool
}

type ProviderCredential struct {
	ProviderName    string    `json:"provider_name"`
	CredentialKey   string    `json:"credential_key"`
	CredentialValue string    `json:"credential_value"`
	UpdatedAt       time.Time `json:"updated_at"`
}
