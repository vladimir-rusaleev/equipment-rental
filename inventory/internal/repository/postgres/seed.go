package postgres

import (
	"context"
	"database/sql"
	"errors"
	"rent/internal/platform"
)

func (r *repository) Seed(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	err = tx.QueryRowContext(ctx, "INSERT INTO seed_history(name) VALUES('demo-v1') ON CONFLICT DO NOTHING RETURNING name").Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	models := []struct {
		id, name, category, description string
		price                           int64
	}{
		{"demo-bosch-gbh226", "Bosch GBH 2-26", "rotary-hammers", "Перфоратор, 800 Вт", 150000},
		{"demo-makita4329", "Makita 4329", "jigsaws", "Электрический лобзик", 80000},
	}
	for _, m := range models {
		if _, err = tx.ExecContext(ctx, "INSERT INTO models(id,name,category,description,daily_price) VALUES($1,$2,$3,$4,$5)", m.id, m.name, m.category, m.description, m.price); err != nil {
			return err
		}
		for n := 1; n <= 3; n++ {
			if _, err = tx.ExecContext(ctx, "INSERT INTO units(id,model_id,state) VALUES($1,$2,'ready')", platform.NewID(), m.id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
