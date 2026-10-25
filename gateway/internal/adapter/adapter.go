package adapter

import (
	"context"
	"encoding/json"
	"net/url"

	pb "rent/pkg/generated"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type backend struct {
	rental    pb.RentalClient
	inventory pb.InventoryClient
}

func New(rental pb.RentalClient, inventory pb.InventoryClient) *backend {
	return &backend{rental: rental, inventory: inventory}
}

func (b *backend) Authenticate(ctx context.Context, token string) (*pb.User, error) {
	return b.rental.Authenticate(ctx, &pb.Token{Token: token})
}

// Call maps a matched HTTP route to its gRPC method. IDs from the URL override body IDs.
func (b *backend) Call(ctx context.Context, route, id string, query url.Values, raw json.RawMessage) (proto.Message, error) {
	switch route {
	case "POST /api/v1/auth/register":
		return unary(ctx, raw, &pb.Credentials{}, b.rental.Register)
	case "POST /api/v1/auth/login":
		return unary(ctx, raw, &pb.Credentials{}, b.rental.Login)
	case "GET /api/v1/models":
		return b.inventory.List(ctx, &pb.ListRequest{Query: query.Get("query"), Category: query.Get("category"), Start: query.Get("start"), End: query.Get("end")})
	case "GET /api/v1/models/{id}":
		return b.inventory.Get(ctx, &pb.ID{Id: id})
	case "GET /api/v1/models/{id}/calendar":
		return b.inventory.CalendarFor(ctx, &pb.AvailabilityRequest{ModelId: id, Start: query.Get("start"), End: query.Get("end")})
	case "POST /api/v1/models":
		return unary(ctx, raw, &pb.Model{}, b.inventory.AddModel)
	case "PUT /api/v1/models/{id}":
		return unary(ctx, raw, &pb.Model{}, b.inventory.UpdateModel, func(r *pb.Model) { r.Id = id })
	case "GET /api/v1/models/{id}/units":
		return b.inventory.ListUnits(ctx, &pb.ID{Id: id})
	case "POST /api/v1/models/{id}/units":
		return b.inventory.AddUnit(ctx, &pb.Unit{ModelId: id})
	case "PATCH /api/v1/units/{id}":
		return unary(ctx, raw, &pb.Unit{}, b.inventory.SetUnitState, func(r *pb.Unit) { r.Id = id })
	case "PATCH /api/v1/users/{id}/role":
		return unary(ctx, raw, &pb.RoleRequest{}, b.rental.SetRole, func(r *pb.RoleRequest) { r.UserId = id })
	case "GET /api/v1/cart":
		return b.rental.GetCart(ctx, &pb.Empty{})
	case "PUT /api/v1/cart":
		return unary(ctx, raw, &pb.Cart{}, b.rental.SetCart)
	case "POST /api/v1/orders":
		return unary(ctx, raw, &pb.OrderRequest{}, b.rental.Checkout)
	case "GET /api/v1/orders":
		return b.rental.ListOrders(ctx, &pb.Empty{})
	case "GET /api/v1/orders/{id}":
		return b.rental.GetOrder(ctx, &pb.ID{Id: id})
	case "POST /api/v1/orders/{id}/cancel":
		return unary(ctx, raw, &pb.Action{}, b.rental.Cancel, func(r *pb.Action) { r.OrderId = id })
	case "POST /api/v1/orders/{id}/issue":
		return unary(ctx, raw, &pb.Action{}, b.rental.Issue, func(r *pb.Action) { r.OrderId = id })
	case "POST /api/v1/orders/{id}/return":
		return unary(ctx, raw, &pb.Action{}, b.rental.Return, func(r *pb.Action) { r.OrderId = id })
	default:
		return nil, status.Error(codes.NotFound, "route not found")
	}
}

func unary[Request proto.Message, Response proto.Message](
	ctx context.Context,
	raw json.RawMessage,
	request Request,
	call func(context.Context, Request, ...grpc.CallOption) (Response, error),
	configure ...func(Request),
) (proto.Message, error) {
	if err := protojson.Unmarshal(raw, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid JSON body")
	}
	for _, prepare := range configure {
		prepare(request)
	}
	return call(ctx, request)
}
