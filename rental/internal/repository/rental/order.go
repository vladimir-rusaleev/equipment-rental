package rental

import (
	"context"
	"sort"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

func (m *memory) FindRequest(_ context.Context, user, key string) (entity.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[m.requests[user+":"+key]]
	if !ok {
		return o, platform.ErrMissing
	}
	return entity.Clone(o), nil
}

func (m *memory) CreateOrder(_ context.Context, o entity.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := o.UserID + ":" + o.RequestID
	if _, ok := m.requests[key]; ok {
		return platform.ErrConflict
	}
	m.orders[o.ID] = entity.Clone(o)
	m.requests[key] = o.ID
	return nil
}

func (m *memory) GetOrder(_ context.Context, id string) (entity.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[id]
	if !ok {
		return o, platform.ErrMissing
	}
	return entity.Clone(o), nil
}

func (m *memory) SaveOrder(_ context.Context, o entity.Order, event *entity.Notification, clearCart bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.orders[o.ID]; ok && current.Version != o.Version {
		return platform.ErrConflict
	}
	o.Version++
	m.orders[o.ID] = entity.Clone(o)
	if event != nil {
		m.outbox[event.ID] = *event
	}
	if clearCart && m.carts[o.UserID].Version == o.CartVersion {
		c := m.carts[o.UserID]
		c.Items = nil
		c.Version++
		m.carts[o.UserID] = c
	}
	return nil
}

func (m *memory) ListOrders(_ context.Context) ([]entity.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []entity.Order{}
	for _, o := range m.orders {
		out = append(out, entity.Clone(o))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
