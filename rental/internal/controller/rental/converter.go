package rental

import (
	pb "rent/pkg/generated"
	"rent/rental/internal/entity"
)

func user(u entity.User) *pb.User { return &pb.User{Id: u.ID, Email: u.Email, Role: u.Role} }

func session(s entity.Session) *pb.Session { return &pb.Session{Token: s.Token, User: user(s.User)} }

func item(i entity.Item) *pb.Item {
	return &pb.Item{ModelId: i.ModelID, Quantity: i.Quantity, Start: i.Start, End: i.End}
}

func cart(c entity.Cart) *pb.Cart {
	out := &pb.Cart{}
	for _, i := range c.Items {
		out.Items = append(out.Items, item(i))
	}
	return out
}

func order(o entity.Order) *pb.Order {
	out := &pb.Order{Id: o.ID, UserId: o.UserID, State: o.State, Total: o.Total, Error: o.Error}
	for _, q := range o.Items {
		out.Items = append(out.Items, &pb.Quote{Item: item(q.Item), Name: q.Name, DailyPrice: q.DailyPrice, Total: q.Total, UnitIds: q.UnitIDs, ReturnedUnitIds: q.ReturnedUnitIDs})
	}
	for _, event := range o.History {
		out.History = append(out.History, &pb.Event{Action: event.Action, ActorId: event.ActorID, Reason: event.Reason, OccurredAt: event.OccurredAt})
	}
	return out
}
