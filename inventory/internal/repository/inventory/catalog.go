package inventory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
)

func (m *memory) List(_ context.Context, query, category, start, end string) ([]entity.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := []entity.Model{}
	for _, v := range m.models {
		if category != "" && v.Category != category {
			continue
		}
		if !strings.Contains(strings.ToLower(v.Name+" "+v.Description), strings.ToLower(query)) {
			continue
		}
		if start != "" {
			v.Available = int32(len(m.available(v.ID, start, end)))
			if v.Available == 0 {
				continue
			}
		}
		v.ImageURLs = append([]string(nil), v.ImageURLs...)
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (m *memory) Get(_ context.Context, id string) (entity.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.models[id]
	if !ok {
		return v, platform.ErrMissing
	}
	v.ImageURLs = append([]string(nil), v.ImageURLs...)
	return v, nil
}

func (m *memory) AddModel(_ context.Context, v entity.Model) (entity.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v.ID = platform.NewID()
	v.ImageURLs = append([]string(nil), v.ImageURLs...)
	m.models[v.ID] = v
	v.ImageURLs = append([]string(nil), v.ImageURLs...)
	return v, nil
}

func (m *memory) UpdateModel(_ context.Context, v entity.Model) (entity.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.models[v.ID]; !ok {
		return v, platform.ErrMissing
	}
	v.ImageURLs = append([]string(nil), v.ImageURLs...)
	m.models[v.ID] = v
	v.ImageURLs = append([]string(nil), v.ImageURLs...)
	return v, nil
}

func (m *memory) ListUnits(_ context.Context, id string) ([]entity.Unit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.models[id]; !ok {
		return nil, platform.ErrMissing
	}
	out := []entity.Unit{}
	for _, u := range m.units {
		if u.ModelID == id {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *memory) AddUnit(_ context.Context, v entity.Unit) (entity.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.models[v.ModelID]; !ok {
		return v, platform.ErrMissing
	}
	v.ID = platform.NewID()
	v.State = entity.UnitReady
	m.units[v.ID] = v
	return v, nil
}

func (m *memory) SetUnitState(_ context.Context, id, state string) (entity.Unit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.units[id]
	if !ok {
		return u, platform.ErrMissing
	}
	if state != entity.UnitReady {
		for _, r := range m.reservations {
			if r.State == entity.ReservationCancelled || r.State == entity.ReservationCompleted {
				continue
			}
			for _, q := range r.Items {
				for _, uid := range q.UnitIDs {
					if uid == id && !contains(q.ReturnedUnitIDs, id) {
						return u, fmt.Errorf("%w: unit has outstanding reservations", platform.ErrConflict)
					}
				}
			}
		}
	}
	u.State = state
	m.units[id] = u
	return u, nil
}
