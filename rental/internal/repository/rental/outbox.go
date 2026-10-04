package rental

import (
	"context"

	"rent/rental/internal/entity"
)

func (m *memory) Notifications(_ context.Context) ([]entity.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []entity.Notification{}
	for _, n := range m.outbox {
		out = append(out, n)
	}
	return out, nil
}

func (m *memory) AckNotification(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.outbox, id)
	return nil
}
