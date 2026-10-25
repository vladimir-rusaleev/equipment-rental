package grpc

import (
	"context"

	"rent/internal/platform"
	pb "rent/pkg/generated"
	"rent/rental/internal/entity"

	"google.golang.org/grpc"
)

type client struct{ remote pb.InventoryClient }

func New(c grpc.ClientConnInterface) *client { return &client{remote: pb.NewInventoryClient(c)} }

func decode(r *pb.Reservation, err error) (entity.Reservation, error) {
	if err != nil {
		return entity.Reservation{}, platform.DomainError(err)
	}
	out := entity.Reservation{OrderID: r.OrderId, State: r.State}
	for _, q := range r.Items {
		out.Items = append(out.Items, entity.Quote{Item: entity.Item{ModelID: q.Item.ModelId, Quantity: q.Item.Quantity, Start: q.Item.Start, End: q.Item.End}, Name: q.Name, DailyPrice: q.DailyPrice, Total: q.Total, UnitIDs: q.UnitIds, ReturnedUnitIDs: q.ReturnedUnitIds})
	}
	return out, nil
}

func (c *client) Reserve(ctx context.Context, id string, items []entity.Item) (entity.Reservation, error) {
	r := &pb.ReserveRequest{OrderId: id}
	for _, i := range items {
		r.Items = append(r.Items, &pb.Item{ModelId: i.ModelID, Quantity: i.Quantity, Start: i.Start, End: i.End})
	}
	v, err := c.remote.Reserve(platform.InternalContext(ctx), r)
	return decode(v, err)
}

func (c *client) Cancel(ctx context.Context, id string) (entity.Reservation, error) {
	v, err := c.remote.Cancel(platform.InternalContext(ctx), &pb.ID{Id: id})
	return decode(v, err)
}

func (c *client) Issue(ctx context.Context, id string) (entity.Reservation, error) {
	v, err := c.remote.Issue(platform.InternalContext(ctx), &pb.ID{Id: id})
	return decode(v, err)
}

func (c *client) Return(ctx context.Context, id string, units []string) (entity.Reservation, error) {
	v, err := c.remote.Return(platform.InternalContext(ctx), &pb.ReturnRequest{OrderId: id, UnitIds: units})
	return decode(v, err)
}
