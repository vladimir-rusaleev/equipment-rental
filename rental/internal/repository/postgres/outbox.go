package postgres

import (
	"context"
	"rent/rental/internal/entity"
)

func (r *repository) Notifications(ctx context.Context) ([]entity.Notification, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,user_id,order_id,state FROM outbox ORDER BY created_at,id LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []entity.Notification{}
	for rows.Next() {
		var n entity.Notification
		if err = rows.Scan(&n.ID, &n.UserID, &n.OrderID, &n.State); err != nil {
			return nil, err
		}
		events = append(events, n)
	}
	return events, rows.Err()
}
func (r *repository) AckNotification(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM outbox WHERE id=$1", id)
	return err
}
