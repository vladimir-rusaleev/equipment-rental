package cart

import (
	"context"
	"time"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

type cartRepository interface {
	GetCart(context.Context, string) (entity.Cart, error)
	SetCart(context.Context, string, []entity.Item) (entity.Cart, error)
}
type userService interface {
	CurrentUser(context.Context) (entity.User, error)
}
type service struct {
	repo  cartRepository
	users userService
}

func New(repo cartRepository, users userService) *service { return &service{repo: repo, users: users} }

func (s *service) GetCart(ctx context.Context) (entity.Cart, error) {
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return entity.Cart{}, err
	}
	return s.repo.GetCart(ctx, u.ID)
}

func validate(items []entity.Item) error {
	if len(items) > 50 {
		return platform.ErrInvalid
	}
	for _, i := range items {
		a, err := time.Parse(time.DateOnly, i.Start)
		if err != nil {
			return platform.ErrInvalid
		}
		b, err := time.Parse(time.DateOnly, i.End)
		if err != nil || !b.After(a) || b.Sub(a) > 366*24*time.Hour || i.Quantity < 1 || i.Quantity > 100 || i.ModelID == "" {
			return platform.ErrInvalid
		}
	}
	return nil
}

func (s *service) SetCart(ctx context.Context, items []entity.Item) (entity.Cart, error) {
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return entity.Cart{}, err
	}
	if err = validate(items); err != nil {
		return entity.Cart{}, err
	}
	return s.repo.SetCart(ctx, u.ID, items)
}
