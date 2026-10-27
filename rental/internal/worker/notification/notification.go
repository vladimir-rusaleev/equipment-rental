package notification

import (
	"context"
	"log/slog"
	"time"

	"rent/internal/metrics"
	"rent/rental/internal/entity"
)

type outboxRepository interface {
	PendingCount(context.Context) (int, error)
	Notifications(context.Context) ([]entity.Notification, error)
	AckNotification(context.Context, string) error
}
type eventPublisher interface {
	Notify(context.Context, entity.Notification) error
}
type worker struct {
	outbox    outboxRepository
	publisher eventPublisher
}

func New(outbox outboxRepository, publisher eventPublisher) *worker {
	return &worker{outbox: outbox, publisher: publisher}
}

func (w *worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.deliver(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("deliver notifications", "error", err)
			}
		}
	}
}

func (w *worker) deliver(ctx context.Context) error {
	defer func() {
		if count, err := w.outbox.PendingCount(ctx); err == nil {
			metrics.SetQueueDepth("outbox", count)
		}
	}()

	events, err := w.outbox.Notifications(ctx)
	if err != nil {
		return err
	}
	for _, event := range events {
		sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = w.publisher.Notify(sendCtx, event)
		cancel()
		if err != nil {
			metrics.NotificationAttempt("outbox", err)
			return err
		}
		err = w.outbox.AckNotification(ctx, event.ID)
		metrics.NotificationAttempt("outbox", err)
		if err != nil {
			return err
		}
	}
	return nil
}
