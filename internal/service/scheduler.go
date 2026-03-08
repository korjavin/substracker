package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	s.logger.Debug("polling quota for providers")
	for _, p := range s.providers {
		info, err := p.FetchUsageInfo(ctx)
		if err != nil {
			if errors.Is(err, provider.ErrUnauthorized) {
				s.logger.Debug("provider unauthorized, skipping quota poll", "provider", p.Name())
				continue
			}
			s.logger.Error("scheduler: fetch usage info", "provider", p.Name(), "error", err)
			continue
		}

		// Get last state
		lastUsage, err := s.repo.GetProviderUsage(ctx, p.Name())
		var wasBlocked bool
		if err == nil {
			wasBlocked = lastUsage.IsBlocked
		} else if err != sql.ErrNoRows {
			s.logger.Error("scheduler: get provider usage", "error", err)
			// Continue with assuming it wasn't blocked
		}

		// State transition detection
		if wasBlocked && !info.IsBlocked {
			msg := fmt.Sprintf("Your %s quota is unblocked! You can use it again.", p.Name())
			s.notif.SendAll(ctx, 0, 0, msg)
		} else if !wasBlocked && info.IsBlocked {
			// Optional: Notify when blocked
			msg := fmt.Sprintf("Your %s quota has been reached. You will be notified when it unblocks.", p.Name())
			s.notif.SendAll(ctx, 0, 0, msg)
		}

		// Save new state (only after checking transitions)
		err = s.repo.UpsertProviderUsage(ctx, repository.UpsertProviderUsageParams{
			ProviderName:        p.Name(),
			CurrentUsageSeconds: info.CurrentUsageSeconds,
			TotalLimitSeconds:   info.TotalLimitSeconds,
			IsBlocked:           info.IsBlocked,
		})
		if err != nil {
			s.logger.Error("scheduler: upsert provider usage", "error", err)
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
