package order

import (
	"context"
	"errors"
	"time"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

func (s *service) advance(ctx context.Context, o entity.Order) (entity.Order, error) {
	if o.PendingAction == "" {
		return o, nil
	}
	var r entity.Reservation
	var err error
	switch o.PendingAction {
	case entity.ActionReserve:
		items := []entity.Item{}
		for _, q := range o.Items {
			items = append(items, q.Item)
		}
		r, err = s.inventory.Reserve(ctx, o.ID, items)
	case entity.ActionCancel:
		r, err = s.inventory.Cancel(ctx, o.ID)
	case entity.ActionIssue:
		r, err = s.inventory.Issue(ctx, o.ID)
	case entity.ActionReturn:
		r, err = s.inventory.Return(ctx, o.ID, o.PendingUnits)
	}
	if err != nil {
		if errors.Is(err, platform.ErrUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return o, nil
		}
		o.Error = err.Error()
		switch o.PendingAction {
		case entity.ActionReserve:
			o.State = entity.OrderRejected
		case entity.ActionCancel, entity.ActionIssue:
			o.State = entity.OrderConfirmed
		case entity.ActionReturn:
			o.State = entity.OrderActive
		}
		o.PendingAction = ""
		if se := s.repo.SaveOrder(ctx, o, nil, false); se != nil {
			return o, se
		}
		o.Version++
		return o, err
	}
	action := o.PendingAction
	o.Items = r.Items
	o.Total = 0
	for _, q := range o.Items {
		o.Total += q.Total
	}
	o.State = map[string]string{entity.ReservationReserved: entity.OrderConfirmed, entity.OrderCancelled: entity.OrderCancelled, entity.OrderActive: entity.OrderActive, entity.OrderCompleted: entity.OrderCompleted}[r.State]
	o.Error = ""
	o.History = append(o.History, entity.Event{Action: action, ActorID: o.PendingActor, Reason: o.PendingReason, OccurredAt: time.Now().UTC().Format(time.RFC3339Nano)})
	o.PendingAction = ""
	o.PendingActor = ""
	o.PendingReason = ""
	o.PendingUnits = nil
	n := entity.Notification{ID: platform.NewID(), UserID: o.UserID, OrderID: o.ID, State: o.State}
	if err = s.repo.SaveOrder(ctx, o, &n, action == entity.ActionReserve); err != nil {
		return o, err
	}
	o.Version++
	return o, nil
}

// ResumePending continues operations whose response was lost or dependency was unavailable.
func (s *service) ResumePending(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	orders, err := s.repo.ListOrders(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, order := range orders {
		if order.PendingAction != "" {
			operationCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			_, err = s.advance(operationCtx, order)
			cancel()
			if err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
