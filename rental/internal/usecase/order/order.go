package order

import (
	"context"
	"errors"
	"sync"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

type orderRepository interface {
	FindRequest(context.Context, string, string) (entity.Order, error)
	CreateOrder(context.Context, entity.Order) error
	GetOrder(context.Context, string) (entity.Order, error)
	SaveOrder(context.Context, entity.Order, *entity.Notification, bool) error
	ListOrders(context.Context) ([]entity.Order, error)
}
type cartRepository interface {
	GetCart(context.Context, string) (entity.Cart, error)
}
type userService interface {
	CurrentUser(context.Context) (entity.User, error)
}
type inventoryClient interface {
	Reserve(context.Context, string, []entity.Item) (entity.Reservation, error)
	Cancel(context.Context, string) (entity.Reservation, error)
	Issue(context.Context, string) (entity.Reservation, error)
	Return(context.Context, string, []string) (entity.Reservation, error)
}
type service struct {
	mu        sync.Mutex
	repo      orderRepository
	carts     cartRepository
	users     userService
	inventory inventoryClient
}

func New(repo orderRepository, carts cartRepository, users userService, inventory inventoryClient) *service {
	return &service{repo: repo, carts: carts, users: users, inventory: inventory}
}

func (s *service) Checkout(ctx context.Context, key string) (entity.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return entity.Order{}, err
	}
	if key == "" || len(key) > 128 {
		return entity.Order{}, platform.ErrInvalid
	}
	o, err := s.repo.FindRequest(ctx, u.ID, key)
	if err == nil {
		return s.advance(ctx, o)
	}
	if !errors.Is(err, platform.ErrMissing) {
		return o, err
	}
	cart, err := s.carts.GetCart(ctx, u.ID)
	if err != nil {
		return o, err
	}
	if len(cart.Items) == 0 {
		return o, platform.ErrInvalid
	}
	o = entity.Order{ID: platform.NewID(), UserID: u.ID, State: entity.OrderPending, RequestID: key, CartVersion: cart.Version, PendingAction: entity.ActionReserve, PendingActor: u.ID}
	for _, i := range cart.Items {
		o.Items = append(o.Items, entity.Quote{Item: i})
	}
	if err = s.repo.CreateOrder(ctx, o); err != nil {
		if !errors.Is(err, platform.ErrConflict) {
			return o, err
		}
		o, err = s.repo.FindRequest(ctx, u.ID, key)
		if err != nil {
			return o, err
		}
	}
	return s.advance(ctx, o)
}

func (s *service) GetOrder(ctx context.Context, id string) (entity.Order, error) {
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return entity.Order{}, err
	}
	o, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return o, err
	}
	if o.UserID != u.ID && !u.IsStaff() {
		return entity.Order{}, platform.ErrForbidden
	}
	return o, nil
}

func (s *service) ListOrders(ctx context.Context) ([]entity.Order, error) {
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	all, err := s.repo.ListOrders(ctx)
	out := []entity.Order{}
	for _, o := range all {
		if u.IsStaff() || o.UserID == u.ID {
			out = append(out, o)
		}
	}
	return out, err
}
