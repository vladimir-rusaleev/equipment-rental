package order_test

import (
	"context"
	"errors"
	"testing"

	"rent/internal/platform"
	"rent/rental/internal/entity"
	rentalrepository "rent/rental/internal/repository/rental"
	cartusecase "rent/rental/internal/usecase/cart"
	orderusecase "rent/rental/internal/usecase/order"
	userusecase "rent/rental/internal/usecase/user"
)

type fakeInventory struct {
	calls        int
	failFirst    bool
	reservations map[string]entity.Reservation
}

func (f *fakeInventory) Reserve(_ context.Context, id string, items []entity.Item) (entity.Reservation, error) {
	f.calls++
	r, ok := f.reservations[id]
	if !ok {
		r = entity.Reservation{OrderID: id, State: "reserved"}
		for _, i := range items {
			r.Items = append(r.Items, entity.Quote{Item: i, Name: "Drill", DailyPrice: 100, Total: 200, UnitIDs: []string{"unit"}})
		}
		f.reservations[id] = r
	}
	if f.failFirst && f.calls == 1 {
		return entity.Reservation{}, platform.ErrUnavailable
	}
	return r, nil
}

func (f *fakeInventory) Cancel(_ context.Context, id string) (entity.Reservation, error) {
	r := f.reservations[id]
	r.State = "cancelled"
	f.reservations[id] = r
	return r, nil
}

func (f *fakeInventory) Issue(_ context.Context, id string) (entity.Reservation, error) {
	r := f.reservations[id]
	r.State = "active"
	return r, nil
}

func (f *fakeInventory) Return(_ context.Context, id string, _ []string) (entity.Reservation, error) {
	r := f.reservations[id]
	r.State = "completed"
	return r, nil
}

func actor(id string) context.Context { return platform.WithActor(context.Background(), id, "") }

func TestLostReserveResponseAndAuthorization(t *testing.T) {
	repo := rentalrepository.NewMemory()
	f := &fakeInventory{failFirst: true, reservations: map[string]entity.Reservation{}}
	users := userusecase.New(repo)
	carts := cartusecase.New(repo, users)
	s := orderusecase.New(repo, repo, users, f)
	u, _ := repo.CreateUser(context.Background(), entity.User{Email: "a@example.com", Role: "customer"})
	other, _ := repo.CreateUser(context.Background(), entity.User{Email: "b@example.com", Role: "customer"})
	ctx := actor(u.ID)
	items := []entity.Item{{ModelID: "drill", Quantity: 1, Start: "2026-11-10", End: "2026-11-12"}}
	if _, err := carts.SetCart(ctx, items); err != nil {
		t.Fatal(err)
	}
	o, err := s.Checkout(ctx, "request")
	if err != nil || o.State != "pending" {
		t.Fatalf("lost response: %+v %v", o, err)
	}
	o, err = s.Checkout(ctx, "request")
	if err != nil || o.State != "confirmed" || len(f.reservations) != 1 {
		t.Fatal("retry failed", err)
	}
	if _, err = s.GetOrder(actor(other.ID), o.ID); !errors.Is(err, platform.ErrForbidden) {
		t.Fatal("other customer accessed order")
	}
	if _, err = s.Issue(ctx, o.ID, ""); !errors.Is(err, platform.ErrForbidden) {
		t.Fatal("customer issued equipment")
	}
	c, _ := carts.GetCart(ctx)
	if len(c.Items) != 0 {
		t.Fatal("checked out cart not cleared")
	}
	again, err := s.Checkout(ctx, "request")
	if err != nil || again.ID != o.ID {
		t.Fatal("checkout retry changed order")
	}
}

func TestResumePendingAfterLostResponse(t *testing.T) {
	repo := rentalrepository.NewMemory()
	inventory := &fakeInventory{failFirst: true, reservations: map[string]entity.Reservation{}}
	users := userusecase.New(repo)
	carts := cartusecase.New(repo, users)
	orders := orderusecase.New(repo, repo, users, inventory)
	user, err := repo.CreateUser(context.Background(), entity.User{Role: "customer"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := actor(user.ID)
	_, err = carts.SetCart(ctx, []entity.Item{{ModelID: "drill", Quantity: 1, Start: "2026-11-10", End: "2026-11-12"}})
	if err != nil {
		t.Fatal(err)
	}
	order, err := orders.Checkout(ctx, "resume")
	if err != nil || order.State != "pending" {
		t.Fatalf("checkout: %+v %v", order, err)
	}
	if err = orders.ResumePending(ctx); err != nil {
		t.Fatal(err)
	}
	order, err = orders.GetOrder(ctx, order.ID)
	if err != nil || order.State != "confirmed" || len(inventory.reservations) != 1 {
		t.Fatalf("resume: %+v %v", order, err)
	}
	events, err := repo.Notifications(ctx)
	if err != nil || len(events) != 1 {
		t.Fatalf("outbox: %+v %v", events, err)
	}
	cart, err := carts.GetCart(ctx)
	if err != nil || len(cart.Items) != 0 {
		t.Fatalf("cart not cleared: %+v %v", cart, err)
	}
}
