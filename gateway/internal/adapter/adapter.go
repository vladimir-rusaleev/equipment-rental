package adapter

import (
	"context"
	"net/url"

	pb "rent/pkg/generated"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type backend struct {
	inventory pb.InventoryClient
}

func New(inventory pb.InventoryClient) *backend {
	return &backend{inventory: inventory}
}

func (b *backend) Call(ctx context.Context, route, id string, query url.Values) (proto.Message, error) {
	switch route {
	case "GET /api/v1/models":
		return b.inventory.List(ctx, &pb.ListRequest{Query: query.Get("query"), Category: query.Get("category"), Start: query.Get("start"), End: query.Get("end")})
	case "GET /api/v1/models/{id}":
		return b.inventory.Get(ctx, &pb.ID{Id: id})
	case "GET /api/v1/models/{id}/calendar":
		return b.inventory.CalendarFor(ctx, &pb.AvailabilityRequest{ModelId: id, Start: query.Get("start"), End: query.Get("end")})
	default:
		return nil, status.Error(codes.NotFound, "route not found")
	}
}
