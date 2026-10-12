package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"rent/internal/platform"
	"rent/notifier/internal/entity"
)

func TestDurableDeduplicationAndPendingDelivery(t *testing.T) {
	dsn := os.Getenv("NOTIFIER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set NOTIFIER_TEST_DATABASE_URL to a disposable test database")
	}
	ctx := context.Background()
	repo, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	event := entity.Notification{ID: platform.NewID(), UserID: "user", OrderID: "order", State: "confirmed"}
	if err = repo.Put(ctx, event); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err = reopened.Put(ctx, event); err != nil {
		t.Fatal(err)
	}
	events, err := reopened.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, e := range events {
		if e.ID == event.ID {
			found++
		}
	}
	if found != 1 {
		t.Fatal("notification lost or duplicated")
	}
	if err = reopened.Sent(ctx, event.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Put(ctx, event); err != nil {
		t.Fatal(err)
	}
	events, err = repo.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.ID == event.ID {
			t.Fatal("sent notification enqueued again")
		}
	}
	event.State = "cancelled"
	if err = repo.Put(ctx, event); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("event ID reused with different payload", err)
	}
}
