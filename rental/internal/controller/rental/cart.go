package rental

import (
	"context"

	"rent/internal/platform"
	pb "rent/pkg/generated"
	"rent/rental/internal/entity"
)

func (c *controller) GetCart(ctx context.Context, _ *pb.Empty) (*pb.Cart, error) {
	v, err := c.carts.GetCart(ctx)
	return cart(v), platform.GRPCError(err)
}

func (c *controller) SetCart(ctx context.Context, r *pb.Cart) (*pb.Cart, error) {
	items := []entity.Item{}
	for _, i := range r.Items {
		items = append(items, entity.Item{ModelID: i.ModelId, Quantity: i.Quantity, Start: i.Start, End: i.End})
	}
	v, err := c.carts.SetCart(ctx, items)
	return cart(v), platform.GRPCError(err)
}
