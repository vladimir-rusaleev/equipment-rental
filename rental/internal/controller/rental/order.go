package rental

import (
	"context"

	"rent/internal/platform"
	pb "rent/pkg/generated"
)

func (c *controller) Checkout(ctx context.Context, r *pb.OrderRequest) (*pb.Order, error) {
	v, err := c.orders.Checkout(ctx, r.RequestId)
	return order(v), platform.GRPCError(err)
}

func (c *controller) GetOrder(ctx context.Context, r *pb.ID) (*pb.Order, error) {
	v, err := c.orders.GetOrder(ctx, r.Id)
	return order(v), platform.GRPCError(err)
}

func (c *controller) ListOrders(ctx context.Context, _ *pb.Empty) (*pb.Orders, error) {
	v, err := c.orders.ListOrders(ctx)
	out := &pb.Orders{}
	for _, o := range v {
		out.Items = append(out.Items, order(o))
	}
	return out, platform.GRPCError(err)
}

func (c *controller) Cancel(ctx context.Context, r *pb.Action) (*pb.Order, error) {
	v, err := c.orders.Cancel(ctx, r.OrderId, r.Reason)
	return order(v), platform.GRPCError(err)
}

func (c *controller) Issue(ctx context.Context, r *pb.Action) (*pb.Order, error) {
	v, err := c.orders.Issue(ctx, r.OrderId, r.Reason)
	return order(v), platform.GRPCError(err)
}

func (c *controller) Return(ctx context.Context, r *pb.Action) (*pb.Order, error) {
	v, err := c.orders.Return(ctx, r.OrderId, r.Reason, r.UnitIds)
	return order(v), platform.GRPCError(err)
}
