package notification

import (
	"context"
	"sync"

	"rent/internal/platform"
	"rent/notifier/internal/entity"
)

type record struct {
	Notification entity.Notification
	Sent         bool
}
type memory struct {
	mu      sync.Mutex
	records map[string]record
}

func NewMemory() *memory { return &memory{records: map[string]record{}} }

func (m *memory) Put(_ context.Context, n entity.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.records[n.ID]; ok {
		if old.Notification != n {
			return platform.ErrConflict
		}
		return nil
	}
	m.records[n.ID] = record{Notification: n}
	return nil
}

func (m *memory) Pending(_ context.Context) ([]entity.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []entity.Notification{}
	for _, r := range m.records {
		if !r.Sent {
			out = append(out, r.Notification)
		}
	}
	return out, nil
}

func (m *memory) Sent(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.records[id]
	r.Sent = true
	m.records[id] = r
	return nil
}
