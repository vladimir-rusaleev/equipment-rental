package grpc

import (
	"context"

	"rent/internal/platform"
	pb "rent/pkg/generated"
	"rent/rental/internal/entity"

	"google.golang.org/grpc"
)

type client struct{ remote pb.NotifierClient }

func New(c grpc.ClientConnInterface) *client { return &client{remote: pb.NewNotifierClient(c)} }

func (c *client) Notify(ctx context.Context, n entity.Notification) error {
	_, err := c.remote.Notify(platform.InternalContext(ctx), &pb.Notification{Id: n.ID, UserId: n.UserID, OrderId: n.OrderID, State: n.State})
	return platform.DomainError(err)
}
