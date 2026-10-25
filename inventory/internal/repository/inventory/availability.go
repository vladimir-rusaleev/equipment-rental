package inventory

import (
	"context"
	"sort"
	"time"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
)

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func overlaps(a, b, c, d string) bool { return a < d && c < b }

func (m *memory) available(model, start, end string) []string {
	ids := []string{}
	for id, u := range m.units {
		if u.ModelID != model || u.State != entity.UnitReady {
			continue
		}
		free := true
		for _, r := range m.reservations {
			if r.State == entity.ReservationCancelled {
				continue
			}
			for _, q := range r.Items {
				if !contains(q.UnitIDs, id) {
					continue
				}
				if overlaps(start, end, q.Item.Start, q.Item.End) {
					free = false
				}
				if r.State == entity.ReservationActive && !contains(q.ReturnedUnitIDs, id) && q.Item.End <= time.Now().UTC().Format(time.DateOnly) {
					free = false
				}
			}
		}
		if free {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (m *memory) Calendar(_ context.Context, model, start, end string) ([]entity.Day, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.models[model]; !ok {
		return nil, platform.ErrMissing
	}
	a, b, _ := entity.Period(start, end)
	days := []entity.Day{}
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		s := d.Format(time.DateOnly)
		err := d.AddDate(0, 0, 1).Format(time.DateOnly)
		days = append(days, entity.Day{Date: s, Available: int32(len(m.available(model, s, err)))})
	}
	return days, nil
}
