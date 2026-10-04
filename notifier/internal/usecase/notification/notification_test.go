package notification

import (
	"context"
	"errors"
	"testing"

	"rent/internal/platform"
	"rent/notifier/internal/entity"
	notificationrepository "rent/notifier/internal/repository/notification"
)

func TestInboxDeduplicationAndRetryState(t *testing.T) {
	ctx := context.Background()
	repo := notificationrepository.NewMemory()
	event := entity.Notification{ID: "event", UserID: "user", OrderID: "order", State: "confirmed"}
	if err := repo.Put(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := repo.Put(ctx, event); err != nil {
		t.Fatal(err)
	}
	pending, _ := repo.Pending(ctx)
	if len(pending) != 1 {
		t.Fatal("duplicate delivery enqueued twice")
	}
	event.State = "cancelled"
	if err := repo.Put(ctx, event); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("different event accepted with same ID")
	}
	_ = repo.Sent(ctx, "event")
	event.State = "confirmed"
	_ = repo.Put(ctx, event)
	pending, _ = repo.Pending(ctx)
	if len(pending) != 0 {
		t.Fatal("delivered notification enqueued again")
	}
}
