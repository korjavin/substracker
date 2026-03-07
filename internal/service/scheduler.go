package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/korjavin/substracker/internal/repository"
)

type Scheduler struct {
	repo   *repository.Queries
	notif  *NotificationService
	logger *slog.Logger
}

func NewScheduler(repo *repository.Queries, notif *NotificationService, logger *slog.Logger) *Scheduler {
	return &Scheduler{repo: repo, notif: notif, logger: logger}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("scheduler started")

	// Check at startup in case server restarted on billing day
	s.check(ctx)

	for {
		// Sleep until 00:05 of the next day
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

func (s *Scheduler) check(ctx context.Context) {
	now := time.Now()
	today := now.Day()
	tomorrow := now.AddDate(0, 0, 1).Day()

	subs, err := s.repo.ListSubscriptions(ctx)
	if err != nil {
		s.logger.Error("scheduler: list subscriptions", "error", err)
		return
	}

	for _, sub := range subs {
		day := int(sub.BillingDay)
		switch {
		case day == today:
			msg := fmt.Sprintf("Your %s subscription has reset! New billing cycle started.", sub.Name)
			s.notif.SendAll(ctx, sub.ID, msg)
		case day == tomorrow:
			msg := fmt.Sprintf("Reminder: Your %s subscription resets tomorrow (day %d).", sub.Name, day)
			s.notif.SendAll(ctx, sub.ID, msg)
		}
	}
}
