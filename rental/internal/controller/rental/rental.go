package rental

import (
	"context"

	pb "rent/pkg/generated"
	"rent/rental/internal/entity"
)

type userService interface {
	Register(context.Context, string, string) (entity.Session, error)
	Login(context.Context, string, string) (entity.Session, error)
	Authenticate(context.Context, string) (entity.User, error)
	SetRole(context.Context, string, string) (entity.User, error)
}
type cartService interface {
	GetCart(context.Context) (entity.Cart, error)
	SetCart(context.Context, []entity.Item) (entity.Cart, error)
}
type orderService interface {
	Checkout(context.Context, string) (entity.Order, error)
	GetOrder(context.Context, string) (entity.Order, error)
	ListOrders(context.Context) ([]entity.Order, error)
	Cancel(context.Context, string, string) (entity.Order, error)
	Issue(context.Context, string, string) (entity.Order, error)
	Return(context.Context, string, string, []string) (entity.Order, error)
}
type controller struct {
	pb.UnimplementedRentalServer
	users  userService
	carts  cartService
	orders orderService
}

func New(users userService, carts cartService, orders orderService) *controller {
	return &controller{users: users, carts: carts, orders: orders}
}
