package notification

import (
	"context"
	"log/slog"
	"time"

	"rent/rental/internal/entity"
)

type outboxRepository interface {
	Notifications(context.Context) ([]entity.Notification, error)
	AckNotification(context.Context, string) error
}
type notifierClient interface {
	Notify(context.Context, entity.Notification) error
}
type worker struct {
	outbox   outboxRepository
	notifier notifierClient
}

func New(outbox outboxRepository, notifier notifierClient) *worker {
	return &worker{outbox: outbox, notifier: notifier}
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
	events, err := w.outbox.Notifications(ctx)
	if err != nil {
		return err
	}
	for _, event := range events {
		sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = w.notifier.Notify(sendCtx, event)
		cancel()
		if err != nil {
			continue
		}
		if err = w.outbox.AckNotification(ctx, event.ID); err != nil {
			return err
		}
	}
	return nil
}
