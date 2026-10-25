package inventory

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
)

func (m *memory) Reserve(_ context.Context, id string, items []entity.Item) (entity.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.reservations[id]; ok {
		if !reflect.DeepEqual(m.requests[id], items) {
			return entity.Reservation{}, platform.ErrConflict
		}
		return entity.Clone(r), nil
	}
	r := entity.Reservation{OrderID: id, State: entity.ReservationReserved, Items: []entity.Quote{}}
	for _, i := range items {
		model, ok := m.models[i.ModelID]
		if !ok {
			return entity.Reservation{}, platform.ErrMissing
		}
		candidates := m.available(i.ModelID, i.Start, i.End)
		chosen := []string{}
		for _, uid := range candidates {
			used := false
			for _, q := range r.Items {
				if contains(q.UnitIDs, uid) && overlaps(i.Start, i.End, q.Item.Start, q.Item.End) {
					used = true
				}
			}
			if !used {
				chosen = append(chosen, uid)
			}
			if len(chosen) == int(i.Quantity) {
				break
			}
		}
		if len(chosen) < int(i.Quantity) {
			return entity.Reservation{}, fmt.Errorf("%w: insufficient equipment for %s", platform.ErrConflict, i.ModelID)
		}
		a, b, _ := entity.Period(i.Start, i.End)
		days := int64(b.Sub(a) / (24 * time.Hour))
		r.Items = append(r.Items, entity.Quote{Item: i, Name: model.Name, DailyPrice: model.DailyPrice, Total: days * int64(i.Quantity) * model.DailyPrice, UnitIDs: chosen})
	}
	m.reservations[id] = r
	m.requests[id] = append([]entity.Item(nil), items...)
	return entity.Clone(r), nil
}

func (m *memory) GetReservation(_ context.Context, id string) (entity.Reservation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.reservations[id]
	if !ok {
		return r, platform.ErrMissing
	}
	return entity.Clone(r), nil
}

func (m *memory) Cancel(_ context.Context, id string) (entity.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reservations[id]
	if !ok {
		return r, platform.ErrMissing
	}
	if r.State != entity.ReservationReserved && r.State != entity.ReservationCancelled {
		return r, platform.ErrConflict
	}
	r.State = entity.ReservationCancelled
	m.reservations[id] = r
	return entity.Clone(r), nil
}

func (m *memory) Issue(_ context.Context, id string) (entity.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reservations[id]
	if !ok {
		return r, platform.ErrMissing
	}
	if r.State == entity.ReservationActive {
		return entity.Clone(r), nil
	}
	if r.State != entity.ReservationReserved {
		return r, platform.ErrConflict
	}
	for _, q := range r.Items {
		for _, uid := range q.UnitIDs {
			if m.units[uid].State != entity.UnitReady {
				return r, platform.ErrConflict
			}
			for oid, other := range m.reservations {
				if oid == id || other.State != entity.ReservationActive {
					continue
				}
				for _, oq := range other.Items {
					if contains(oq.UnitIDs, uid) && !contains(oq.ReturnedUnitIDs, uid) {
						return r, fmt.Errorf("%w: unit not physically returned", platform.ErrConflict)
					}
				}
			}
		}
	}
	r.State = entity.ReservationActive
	m.reservations[id] = r
	return entity.Clone(r), nil
}

func (m *memory) Return(_ context.Context, id string, ids []string) (entity.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reservations[id]
	if !ok {
		return r, platform.ErrMissing
	}
	if r.State != entity.ReservationActive && r.State != entity.ReservationCompleted {
		return r, platform.ErrConflict
	}
	if len(ids) == 0 {
		return r, platform.ErrInvalid
	}
	for _, id := range ids {
		found := false
		for _, q := range r.Items {
			if contains(q.UnitIDs, id) {
				found = true
			}
		}
		if !found {
			return entity.Reservation{}, platform.ErrInvalid
		}
	}
	// Build a copy before changing nested slices.
	r = entity.Clone(r)
	all := true
	for i := range r.Items {
		q := &r.Items[i]
		for _, id := range ids {
			if contains(q.UnitIDs, id) && !contains(q.ReturnedUnitIDs, id) {
				q.ReturnedUnitIDs = append(q.ReturnedUnitIDs, id)
			}
		}
		if len(q.UnitIDs) != len(q.ReturnedUnitIDs) {
			all = false
		}
	}
	if all {
		r.State = entity.ReservationCompleted
	}
	m.reservations[id] = r
	return entity.Clone(r), nil
}
