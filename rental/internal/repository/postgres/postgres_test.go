package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

func testRepository(t *testing.T) *repository {
	t.Helper()
	dsn := os.Getenv("RENTAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set RENTAL_TEST_DATABASE_URL to a disposable test database")
	}
	repo, err := New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}
func TestOrderCartOutboxAtomicityAndVersion(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	user, err := repo.CreateUser(ctx, entity.User{Email: platform.NewID() + "@example.com", PasswordHash: "test", Role: entity.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	cart, err := repo.SetCart(ctx, user.ID, []entity.Item{{ModelID: "drill", Quantity: 1, Start: "2030-11-10", End: "2030-11-12"}})
	if err != nil {
		t.Fatal(err)
	}
	order := entity.Order{ID: platform.NewID(), UserID: user.ID, RequestID: "key", State: entity.OrderPending, CartVersion: cart.Version}
	if err = repo.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}
	order.State = entity.OrderConfirmed
	event := entity.Notification{ID: platform.NewID(), UserID: user.ID, OrderID: order.ID, State: order.State}
	if err = repo.SaveOrder(ctx, order, &event, true); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveOrder(ctx, order, &event, true); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("stale order update accepted", err)
	}
	reopened := testRepository(t)
	stored, err := reopened.GetOrder(ctx, order.ID)
	if err != nil || stored.State != entity.OrderConfirmed || stored.Version != 1 {
		t.Fatal("order did not survive reopen", err)
	}
	cart, err = reopened.GetCart(ctx, user.ID)
	if err != nil || len(cart.Items) != 0 {
		t.Fatal("cart not cleared", err)
	}
	events, err := reopened.Notifications(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.ID == event.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("outbox event missing")
	}
	// Force a failure after the order UPDATE; the complete transaction must roll back.
	stored.State = entity.OrderCancelled
	invalid := event
	invalid.ID = platform.NewID()
	invalid.UserID = "missing-user"
	if err = reopened.SaveOrder(ctx, stored, &invalid, false); err == nil {
		t.Fatal("invalid foreign key accepted")
	}
	unchanged, err := reopened.GetOrder(ctx, order.ID)
	if err != nil || unchanged.State != entity.OrderConfirmed || unchanged.Version != 1 {
		t.Fatal("failed save was not rolled back", err)
	}
}
func TestSessionBootstrapAndNewCartSurvive(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	email := platform.NewID() + "@example.com"
	if err := repo.EnsureAdmin(ctx, email, "original-hash"); err != nil {
		t.Fatal(err)
	}
	if err := repo.EnsureAdmin(ctx, email, "replacement-hash"); err != nil {
		t.Fatal(err)
	}
	admin, err := repo.UserByEmail(ctx, email)
	if err != nil || admin.PasswordHash != "original-hash" {
		t.Fatal("bootstrap reset password", err)
	}
	token := platform.NewID()
	if err = repo.PutSession(ctx, token, admin.ID); err != nil {
		t.Fatal(err)
	}
	reopened := testRepository(t)
	restored, err := reopened.SessionUser(ctx, token)
	if err != nil || restored.ID != admin.ID {
		t.Fatal("session lost", err)
	}
	if _, err = repo.db.ExecContext(ctx, "UPDATE sessions SET expires_at=now()-interval '1 minute' WHERE token_hash=$1", tokenHash(token)); err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.SessionUser(ctx, token); !errors.Is(err, platform.ErrUnauthenticated) {
		t.Fatal("expired token accepted", err)
	}
	old, err := repo.SetCart(ctx, admin.ID, []entity.Item{{ModelID: "old"}})
	if err != nil {
		t.Fatal(err)
	}
	order := entity.Order{ID: platform.NewID(), UserID: admin.ID, RequestID: "cart-test", State: entity.OrderPending, CartVersion: old.Version}
	if err = repo.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SetCart(ctx, admin.ID, []entity.Item{{ModelID: "new"}}); err != nil {
		t.Fatal(err)
	}
	order.State = entity.OrderConfirmed
	if err = repo.SaveOrder(ctx, order, nil, true); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetCart(ctx, admin.ID)
	if err != nil || len(current.Items) != 1 || current.Items[0].ModelID != "new" {
		t.Fatal("newer cart was erased", err)
	}
	customer, err := repo.CreateUser(ctx, entity.User{Email: platform.NewID() + "@example.com", PasswordHash: "hash", Role: entity.RoleCustomer})
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.EnsureAdmin(ctx, customer.Email, "hash"); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("bootstrap elevated customer", err)
	}
}
