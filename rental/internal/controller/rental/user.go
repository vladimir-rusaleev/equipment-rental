package rental

import (
	"context"

	"rent/internal/platform"
	pb "rent/pkg/generated"
)

func (c *controller) Register(ctx context.Context, r *pb.Credentials) (*pb.Session, error) {
	s, err := c.users.Register(ctx, r.Email, r.Password)
	return session(s), platform.GRPCError(err)
}

func (c *controller) Login(ctx context.Context, r *pb.Credentials) (*pb.Session, error) {
	s, err := c.users.Login(ctx, r.Email, r.Password)
	return session(s), platform.GRPCError(err)
}

func (c *controller) Authenticate(ctx context.Context, r *pb.Token) (*pb.User, error) {
	u, err := c.users.Authenticate(ctx, r.Token)
	return user(u), platform.GRPCError(err)
}

func (c *controller) SetRole(ctx context.Context, r *pb.RoleRequest) (*pb.User, error) {
	u, err := c.users.SetRole(ctx, r.UserId, r.Role)
	return user(u), platform.GRPCError(err)
}
