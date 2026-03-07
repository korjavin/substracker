package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/korjavin/substracker/internal/provider"
	"github.com/korjavin/substracker/internal/repository"
)

type Scheduler struct {
	repo         *repository.Queries
	notif        *NotificationService
	logger       *slog.Logger
	providers    []provider.Provider
	pollInterval time.Duration
}

func NewScheduler(repo *repository.Queries, notif *NotificationService, logger *slog.Logger, providers []provider.Provider, pollInterval time.Duration) *Scheduler {
	return &Scheduler{
		repo:         repo,
		notif:        notif,
		logger:       logger,
		providers:    providers,
		pollInterval: pollInterval,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("scheduler started", "poll_interval", s.pollInterval)

	// Run billing check in a separate goroutine
	go s.runBillingCheck(ctx)

	// Run quota poll in this goroutine
	s.runQuotaPoll(ctx)
}

func (s *Scheduler) runBillingCheck(ctx context.Context) {
	// Check at startup in case server restarted on billing day
	s.check(ctx)

	for {
		// Sleep until 00:05 of the next day for billing check
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
		timer := time.NewTimer(time.Until(next))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.check(ctx)
		}
	}
}

func (s *Scheduler) runQuotaPoll(ctx context.Context) {
	if s.pollInterval <= 0 {
		s.logger.Warn("scheduler: pollInterval is non-positive, quota polling disabled", "interval", s.pollInterval)
		return
	}

	// Poll quota immediately
	s.pollQuota(ctx)

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollQuota(ctx)
		}
	}
}

func (s *Scheduler) pollQuota(ctx context.Context) {
	s.logger.Debug("polling quota for subscriptions")

	subs, err := s.repo.ListAllSubscriptions(ctx)
	if err != nil {
		s.logger.Error("scheduler: list all subscriptions", "error", err)
		return
	}

	for _, sub := range subs {
		if sub.AuthToken == "" {
			continue
		}

		var p provider.Provider
		for _, provider := range s.providers {
			// e.g. "claude" vs "Claude", "googleone" vs "MockGoogleOne"
			// Wait, the easiest matching is strings.Contains or similar, but
			// we can map it via Name() mapping based on how main passes them: Claude -> claude, OpenAI -> openai, etc.
			// Let's just use a simple switch on Name() mapped to service name
			pName := strings.ToLower(provider.Name())
			if strings.Contains(pName, sub.Service) || strings.Contains(sub.Service, pName) {
				p = provider
				break
			}
			// Fallbacks
			if sub.Service == "googleone" && strings.Contains(pName, "google") {
				p = provider
				break
			}
			if sub.Service == "zai" && strings.Contains(pName, "zai") {
				p = provider
				break
			}
		}

		if p == nil {
			continue
		}

		creds := map[string]string{
			"session_key":    sub.AuthToken,
			"session_token":  sub.AuthToken,
			"session_cookie": sub.AuthToken,
		}

		err := p.Login(ctx, creds)
		if err != nil {
			s.logger.Error("scheduler: provider login for sub", "subID", sub.ID, "error", err)
			continue
		}

		info, err := p.FetchUsageInfo(ctx)
		if err != nil {
			if errors.Is(err, provider.ErrUnauthorized) {
				s.logger.Debug("provider unauthorized, skipping quota poll for sub", "subID", sub.ID, "provider", p.Name())
				continue
			}
			s.logger.Error("scheduler: fetch usage info", "subID", sub.ID, "provider", p.Name(), "error", err)
			continue
		}

		// Get last state
		lastUsage, err := s.repo.GetSubscriptionUsage(ctx, sub.ID)
		var wasBlocked bool
		if err == nil {
			wasBlocked = lastUsage.IsBlocked
		} else if err != sql.ErrNoRows {
			s.logger.Error("scheduler: get subscription usage", "subID", sub.ID, "error", err)
			// Continue with assuming it wasn't blocked
		}

		// State transition detection
		if wasBlocked && !info.IsBlocked {
			msg := fmt.Sprintf("Your %s quota is unblocked! You can use it again.", sub.Name)
			s.notif.SendAll(ctx, sub.UserID, sub.ID, msg)
		} else if !wasBlocked && info.IsBlocked {
			// Optional: Notify when blocked
			msg := fmt.Sprintf("Your %s quota has been reached. You will be notified when it unblocks.", sub.Name)
			s.notif.SendAll(ctx, sub.UserID, sub.ID, msg)
		}

		// Save new state
		err = s.repo.UpsertSubscriptionUsage(ctx, repository.UpsertSubscriptionUsageParams{
			SubscriptionID:      sub.ID,
			CurrentUsageSeconds: info.CurrentUsageSeconds,
			TotalLimitSeconds:   info.TotalLimitSeconds,
			IsBlocked:           info.IsBlocked,
		})
		if err != nil {
			s.logger.Error("scheduler: upsert subscription usage", "subID", sub.ID, "error", err)
			continue
		}
	}
}

func (s *Scheduler) check(ctx context.Context) {
	now := time.Now()
	today := now.Day()
	tomorrow := now.AddDate(0, 0, 1).Day()

	subs, err := s.repo.ListAllSubscriptions(ctx)
	if err != nil {
		s.logger.Error("scheduler: list subscriptions", "error", err)
		return
	}

	for _, sub := range subs {
		day := int(sub.BillingDay)
		switch {
		case day == today:
			msg := fmt.Sprintf("Your %s subscription has reset! New billing cycle started.", sub.Name)
			s.notif.SendAll(ctx, sub.UserID, sub.ID, msg)
		case day == tomorrow:
			msg := fmt.Sprintf("Reminder: Your %s subscription resets tomorrow (day %d).", sub.Name, day)
			s.notif.SendAll(ctx, sub.UserID, sub.ID, msg)
		}
	}
}
