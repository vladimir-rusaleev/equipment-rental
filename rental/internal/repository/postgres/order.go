package postgres

import (
	"context"
	"encoding/json"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/rental/internal/entity"
)

func scanOrder(row scanner) (entity.Order, error) {
	var o entity.Order
	var data []byte
	if err := row.Scan(&data); err != nil {
		return o, database.Error(err)
	}
	err := json.Unmarshal(data, &o)
	return o, err
}
func (r *repository) FindRequest(ctx context.Context, user, key string) (entity.Order, error) {
	return scanOrder(r.db.QueryRowContext(ctx, "SELECT payload FROM orders WHERE user_id=$1 AND request_id=$2", user, key))
}
func (r *repository) GetOrder(ctx context.Context, id string) (entity.Order, error) {
	return scanOrder(r.db.QueryRowContext(ctx, "SELECT payload FROM orders WHERE id=$1", id))
}
func (r *repository) CreateOrder(ctx context.Context, o entity.Order) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO orders(id,user_id,request_id,state,version,payload) VALUES($1,$2,$3,$4,$5,$6::jsonb)", o.ID, o.UserID, o.RequestID, o.State, o.Version, string(data))
	return database.Error(err)
}
func (r *repository) SaveOrder(ctx context.Context, o entity.Order, event *entity.Notification, clearCart bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	expected := o.Version
	o.Version++
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE orders SET state=$2,version=$3,payload=$4::jsonb WHERE id=$1 AND version=$5", o.ID, o.State, o.Version, string(data), expected)
	if err != nil {
		return database.Error(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return platform.ErrConflict
	}
	if event != nil {
		if _, err = tx.ExecContext(ctx, "INSERT INTO outbox(id,user_id,order_id,state) VALUES($1,$2,$3,$4)", event.ID, event.UserID, event.OrderID, event.State); err != nil {
			return database.Error(err)
		}
	}
	if clearCart {
		if _, err = tx.ExecContext(ctx, "UPDATE carts SET items='[]',version=version+1 WHERE user_id=$1 AND version=$2", o.UserID, o.CartVersion); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *repository) ListOrders(ctx context.Context) ([]entity.Order, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT payload FROM orders ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []entity.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
