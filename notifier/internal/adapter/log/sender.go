package log

import (
	"context"
	"log/slog"

	"rent/notifier/internal/entity"
)

type sender struct{}

func New() *sender { return &sender{} }

func (*sender) Send(_ context.Context, n entity.Notification) error {
	slog.Info("notification", "event_id", n.ID, "user_id", n.UserID, "order_id", n.OrderID, "state", n.State)
	return nil
}
