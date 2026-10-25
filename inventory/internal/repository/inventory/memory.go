package inventory

import (
	"sync"

	"rent/inventory/internal/entity"
)

type memory struct {
	mu           sync.RWMutex
	models       map[string]entity.Model
	units        map[string]entity.Unit
	reservations map[string]entity.Reservation
	requests     map[string][]entity.Item
}

func NewMemory() *memory {
	return &memory{models: map[string]entity.Model{}, units: map[string]entity.Unit{}, reservations: map[string]entity.Reservation{}, requests: map[string][]entity.Item{}}
}
