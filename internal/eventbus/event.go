package eventbus

import "rent/internal/platform"

type OrderEvent struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	OrderID string `json:"order_id"`
	State   string `json:"state"`
}

func (e OrderEvent) Validate() error {
	if e.ID == "" || e.UserID == "" || e.OrderID == "" {
		return platform.ErrInvalid
	}
	switch e.State {
	case "confirmed", "cancelled", "active", "completed":
		return nil
	default:
		return platform.ErrInvalid
	}
}
