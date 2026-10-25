package rental

import (
	"context"

	"rent/rental/internal/entity"
)

func (m *memory) GetCart(_ context.Context, id string) (entity.Cart, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c := m.carts[id]
	c.Items = append([]entity.Item(nil), c.Items...)
	return c, nil
}

func (m *memory) SetCart(_ context.Context, id string, items []entity.Item) (entity.Cart, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := entity.Cart{Items: append([]entity.Item(nil), items...), Version: m.carts[id].Version + 1}
	m.carts[id] = c
	c.Items = append([]entity.Item(nil), c.Items...)
	return c, nil
}
