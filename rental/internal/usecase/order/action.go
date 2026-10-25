package order

import (
	"context"
	"fmt"
	"strings"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

func (s *service) Cancel(ctx context.Context, id, reason string) (entity.Order, error) {
	return s.apply(ctx, id, entity.ActionCancel, reason, nil)
}

func (s *service) Issue(ctx context.Context, id, reason string) (entity.Order, error) {
	return s.apply(ctx, id, entity.ActionIssue, reason, nil)
}

func (s *service) Return(ctx context.Context, id, reason string, units []string) (entity.Order, error) {
	return s.apply(ctx, id, entity.ActionReturn, reason, units)
}

func (s *service) apply(ctx context.Context, id, action, reason string, units []string) (entity.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, err := s.users.CurrentUser(ctx)
	if err != nil {
		return entity.Order{}, err
	}
	o, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return o, err
	}
	if action == entity.ActionCancel {
		if o.UserID != u.ID && !u.IsStaff() {
			return entity.Order{}, platform.ErrForbidden
		}
		if o.UserID != u.ID && strings.TrimSpace(reason) == "" {
			return o, fmt.Errorf("%w: cancellation reason required", platform.ErrInvalid)
		}
	} else if !u.IsStaff() {
		return entity.Order{}, platform.ErrForbidden
	}
	if o.PendingAction != "" {
		o, err = s.advance(ctx, o)
		if err != nil || o.PendingAction != "" {
			return o, err
		}
	}

	switch action {
	case entity.ActionCancel:
		if o.State == entity.OrderCancelled {
			return o, nil
		}
		if o.State != entity.OrderConfirmed {
			return o, platform.ErrConflict
		}
	case entity.ActionIssue:
		if o.State == entity.OrderActive {
			return o, nil
		}
		if o.State != entity.OrderConfirmed {
			return o, platform.ErrConflict
		}
	case entity.ActionReturn:
		if o.State != entity.OrderActive && o.State != entity.OrderCompleted {
			return o, platform.ErrConflict
		}
		if len(units) == 0 {
			return o, platform.ErrInvalid
		}
	}

	if action == entity.ActionReturn {
		allReturned := true
		for _, uid := range units {
			found, returned := false, false
			for _, q := range o.Items {
				for _, id := range q.UnitIDs {
					if uid == id {
						found = true
					}
				}
				for _, id := range q.ReturnedUnitIDs {
					if uid == id {
						returned = true
					}
				}
			}
			if !found {
				return o, platform.ErrInvalid
			}
			if !returned {
				allReturned = false
			}
		}
		if allReturned {
			return o, nil
		}
	}
	o.PendingAction = action
	o.PendingActor = u.ID
	o.PendingReason = reason
	o.PendingUnits = append([]string(nil), units...)
	o.State = map[string]string{entity.ActionCancel: entity.OrderCancelling, entity.ActionIssue: entity.OrderIssuing, entity.ActionReturn: entity.OrderReturning}[action]
	if err = s.repo.SaveOrder(ctx, o, nil, false); err != nil {
		return o, err
	}
	o.Version++
	return s.advance(ctx, o)
}
