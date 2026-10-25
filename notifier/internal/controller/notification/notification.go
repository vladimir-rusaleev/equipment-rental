package notification

import (
	"context"

	"rent/internal/platform"
	"rent/notifier/internal/entity"
	pb "rent/pkg/generated"
)

type notificationService interface {
	Notify(context.Context, entity.Notification) error
}
type controller struct {
	pb.UnimplementedNotifierServer
	service notificationService
}

func New(s notificationService) *controller { return &controller{service: s} }

func (c *controller) Notify(ctx context.Context, r *pb.Notification) (*pb.Empty, error) {
	err := c.service.Notify(ctx, entity.Notification{ID: r.Id, UserID: r.UserId, OrderID: r.OrderId, State: r.State})
	return &pb.Empty{}, platform.GRPCError(err)
}
