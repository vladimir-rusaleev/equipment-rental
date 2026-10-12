package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	database "rent/internal/postgres"
	"rent/rental/internal/entity"
)

func (r *repository) GetCart(ctx context.Context, id string) (entity.Cart, error) {
	var cart entity.Cart
	var data []byte
	err := r.db.QueryRowContext(ctx, "SELECT version,items FROM carts WHERE user_id=$1", id).Scan(&cart.Version, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return cart, nil
	}
	if err != nil {
		return cart, err
	}
	err = json.Unmarshal(data, &cart.Items)
	return cart, err
}
func (r *repository) SetCart(ctx context.Context, id string, items []entity.Item) (entity.Cart, error) {
	if items == nil {
		items = []entity.Item{}
	}
	data, err := json.Marshal(items)
	if err != nil {
		return entity.Cart{}, err
	}
	cart := entity.Cart{Items: items}
	err = r.db.QueryRowContext(ctx, `INSERT INTO carts(user_id,version,items) VALUES($1,1,$2::jsonb)
 ON CONFLICT(user_id) DO UPDATE SET version=carts.version+1,items=excluded.items
 RETURNING version`, id, string(data)).Scan(&cart.Version)
	return cart, database.Error(err)
}
