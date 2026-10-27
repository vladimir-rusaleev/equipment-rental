package notification

import (
	"context"
	"errors"
	"testing"

	"rent/rental/internal/entity"
	rentalrepository "rent/rental/internal/repository/rental"
)

type failingNotifier struct{ attempts int }

func (n *failingNotifier) Notify(context.Context, entity.Notification) error {
	n.attempts++
	if n.attempts == 1 {
		return errors.New("temporary failure")
	}
	return nil
}

func TestFailedDeliveryRemainsInOutbox(t *testing.T) {
	ctx := context.Background()
	repo := rentalrepository.NewMemory()
	event := entity.Notification{ID: "event", OrderID: "order", UserID: "user", State: "confirmed"}
	if err := repo.SaveOrder(ctx, entity.Order{ID: "order"}, &event, false); err != nil {
		t.Fatal(err)
	}
	notifier := &failingNotifier{}
	worker := New(repo, notifier)
	if err := worker.deliver(ctx); err == nil {
		t.Fatal("failed publish was not reported")
	}
	events, err := repo.Notifications(ctx)
	if err != nil || len(events) != 1 {
		t.Fatal("failed delivery was lost", err)
	}
	if err := worker.deliver(ctx); err != nil {
		t.Fatal(err)
	}
	events, err = repo.Notifications(ctx)
	if err != nil || len(events) != 0 || notifier.attempts != 2 {
		t.Fatal("successful retry was not acknowledged", err)
	}
}
