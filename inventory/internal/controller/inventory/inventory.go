package inventory

import (
	"context"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
	pb "rent/pkg/generated"
)

type inventoryService interface {
	List(context.Context, string, string, string, string) ([]entity.Model, error)
	Get(context.Context, string) (entity.Model, error)
	AddModel(context.Context, entity.Model) (entity.Model, error)
	UpdateModel(context.Context, entity.Model) (entity.Model, error)
	ListUnits(context.Context, string) ([]entity.Unit, error)
	AddUnit(context.Context, entity.Unit) (entity.Unit, error)
	SetUnitState(context.Context, string, string) (entity.Unit, error)
	Calendar(context.Context, string, string, string) ([]entity.Day, error)
	Reserve(context.Context, string, []entity.Item) (entity.Reservation, error)
	GetReservation(context.Context, string) (entity.Reservation, error)
	Cancel(context.Context, string) (entity.Reservation, error)
	Issue(context.Context, string) (entity.Reservation, error)
	Return(context.Context, string, []string) (entity.Reservation, error)
}
type controller struct {
	pb.UnimplementedInventoryServer
	service inventoryService
}

func New(s inventoryService) *controller { return &controller{service: s} }

func model(v entity.Model) *pb.Model {
	return &pb.Model{Id: v.ID, Name: v.Name, Category: v.Category, Description: v.Description, DailyPrice: v.DailyPrice, ImageUrls: v.ImageURLs, Available: v.Available}
}

func unit(v entity.Unit) *pb.Unit { return &pb.Unit{Id: v.ID, ModelId: v.ModelID, State: v.State} }

func reservation(v entity.Reservation) *pb.Reservation {
	r := &pb.Reservation{OrderId: v.OrderID, State: v.State}
	for _, q := range v.Items {
		r.Items = append(r.Items, &pb.Quote{Item: &pb.Item{ModelId: q.Item.ModelID, Quantity: q.Item.Quantity, Start: q.Item.Start, End: q.Item.End}, Name: q.Name, DailyPrice: q.DailyPrice, Total: q.Total, UnitIds: q.UnitIDs, ReturnedUnitIds: q.ReturnedUnitIDs})
	}
	return r
}

func staff(ctx context.Context, admin bool) error {
	_, role := platform.Actor(ctx)
	if role == entity.RoleAdmin || (!admin && role == entity.RoleStaff) {
		return nil
	}
	return platform.ErrForbidden
}

func (c *controller) List(ctx context.Context, r *pb.ListRequest) (*pb.Models, error) {
	items, err := c.service.List(ctx, r.Query, r.Category, r.Start, r.End)
	if err != nil {
		return nil, platform.GRPCError(err)
	}
	out := &pb.Models{}
	for _, v := range items {
		out.Items = append(out.Items, model(v))
	}
	return out, nil
}

func (c *controller) Get(ctx context.Context, r *pb.ID) (*pb.Model, error) {
	v, err := c.service.Get(ctx, r.Id)
	return model(v), platform.GRPCError(err)
}

func (c *controller) CalendarFor(ctx context.Context, r *pb.AvailabilityRequest) (*pb.Calendar, error) {
	v, err := c.service.Calendar(ctx, r.ModelId, r.Start, r.End)
	out := &pb.Calendar{}
	for _, d := range v {
		out.Days = append(out.Days, &pb.Day{Date: d.Date, Available: d.Available})
	}
	return out, platform.GRPCError(err)
}

func (c *controller) AddModel(ctx context.Context, r *pb.Model) (*pb.Model, error) {
	if err := staff(ctx, true); err != nil {
		return nil, platform.GRPCError(err)
	}
	v, err := c.service.AddModel(ctx, entity.Model{Name: r.Name, Category: r.Category, Description: r.Description, ImageURLs: r.ImageUrls, DailyPrice: r.DailyPrice})
	return model(v), platform.GRPCError(err)
}

func (c *controller) AddUnit(ctx context.Context, r *pb.Unit) (*pb.Unit, error) {
	if err := staff(ctx, false); err != nil {
		return nil, platform.GRPCError(err)
	}
	v, err := c.service.AddUnit(ctx, entity.Unit{ModelID: r.ModelId})
	return unit(v), platform.GRPCError(err)
}

func (c *controller) SetUnitState(ctx context.Context, r *pb.Unit) (*pb.Unit, error) {
	if err := staff(ctx, false); err != nil {
		return nil, platform.GRPCError(err)
	}
	v, err := c.service.SetUnitState(ctx, r.Id, r.State)
	return unit(v), platform.GRPCError(err)
}

func (c *controller) Reserve(ctx context.Context, r *pb.ReserveRequest) (*pb.Reservation, error) {
	items := []entity.Item{}
	for _, i := range r.Items {
		items = append(items, entity.Item{ModelID: i.ModelId, Quantity: i.Quantity, Start: i.Start, End: i.End})
	}
	v, err := c.service.Reserve(ctx, r.OrderId, items)
	return reservation(v), platform.GRPCError(err)
}

func (c *controller) GetReservation(ctx context.Context, r *pb.ID) (*pb.Reservation, error) {
	v, err := c.service.GetReservation(ctx, r.Id)
	return reservation(v), platform.GRPCError(err)
}

func (c *controller) Cancel(ctx context.Context, r *pb.ID) (*pb.Reservation, error) {
	v, err := c.service.Cancel(ctx, r.Id)
	return reservation(v), platform.GRPCError(err)
}

func (c *controller) Issue(ctx context.Context, r *pb.ID) (*pb.Reservation, error) {
	v, err := c.service.Issue(ctx, r.Id)
	return reservation(v), platform.GRPCError(err)
}

func (c *controller) Return(ctx context.Context, r *pb.ReturnRequest) (*pb.Reservation, error) {
	v, err := c.service.Return(ctx, r.OrderId, r.UnitIds)
	return reservation(v), platform.GRPCError(err)
}

func (c *controller) UpdateModel(ctx context.Context, r *pb.Model) (*pb.Model, error) {
	v, err := c.service.UpdateModel(ctx, entity.Model{ID: r.Id, Name: r.Name, Category: r.Category, Description: r.Description, DailyPrice: r.DailyPrice, ImageURLs: r.ImageUrls})
	return model(v), platform.GRPCError(err)
}

func (c *controller) ListUnits(ctx context.Context, r *pb.ID) (*pb.Units, error) {
	items, err := c.service.ListUnits(ctx, r.Id)
	out := &pb.Units{}
	for _, u := range items {
		out.Items = append(out.Items, unit(u))
	}
	return out, platform.GRPCError(err)
}
