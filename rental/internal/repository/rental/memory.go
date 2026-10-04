package rental

import (
	"sync"

	"rent/rental/internal/entity"
)

type memory struct {
	mu       sync.RWMutex
	users    map[string]entity.User
	emails   map[string]string
	sessions map[string]entity.StoredSession
	carts    map[string]entity.Cart
	orders   map[string]entity.Order
	requests map[string]string
	outbox   map[string]entity.Notification
}

func NewMemory() *memory {
	return &memory{users: map[string]entity.User{}, emails: map[string]string{}, sessions: map[string]entity.StoredSession{}, carts: map[string]entity.Cart{}, orders: map[string]entity.Order{}, requests: map[string]string{}, outbox: map[string]entity.Notification{}}
}
