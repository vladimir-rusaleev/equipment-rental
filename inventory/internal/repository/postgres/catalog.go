package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/inventory/internal/entity"
)

const modelColumns = "id,name,category,description,daily_price,image_urls"

func scanModel(row scanner) (entity.Model, error) {
	var model entity.Model
	var images []byte
	if err := row.Scan(&model.ID, &model.Name, &model.Category, &model.Description, &model.DailyPrice, &images); err != nil {
		return model, database.Error(err)
	}
	err := json.Unmarshal(images, &model.ImageURLs)
	return model, err
}
func (r *repository) List(ctx context.Context, query, category, start, end string) ([]entity.Model, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+modelColumns+" FROM models WHERE ($2='' OR category=$2) AND strpos(lower(name||' '||description),lower($1))>0 ORDER BY id", query, category)
	if err != nil {
		return nil, err
	}
	models := []entity.Model{}
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		models = append(models, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []entity.Model{}
	for _, m := range models {
		if start != "" {
			ids, err := available(ctx, r.db, m.ID, start, end)
			if err != nil {
				return nil, err
			}
			m.Available = int32(len(ids))
			if m.Available == 0 {
				continue
			}
		}
		result = append(result, m)
	}
	return result, nil
}
func (r *repository) Get(ctx context.Context, id string) (entity.Model, error) {
	return scanModel(r.db.QueryRowContext(ctx, "SELECT "+modelColumns+" FROM models WHERE id=$1", id))
}
func (r *repository) AddModel(ctx context.Context, m entity.Model) (entity.Model, error) {
	m.ID = platform.NewID()
	if m.ImageURLs == nil {
		m.ImageURLs = []string{}
	}
	data, err := json.Marshal(m.ImageURLs)
	if err != nil {
		return m, err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO models(id,name,category,description,daily_price,image_urls) VALUES($1,$2,$3,$4,$5,$6::jsonb)", m.ID, m.Name, m.Category, m.Description, m.DailyPrice, string(data))
	return m, database.Error(err)
}
func (r *repository) UpdateModel(ctx context.Context, m entity.Model) (entity.Model, error) {
	if m.ImageURLs == nil {
		m.ImageURLs = []string{}
	}
	data, err := json.Marshal(m.ImageURLs)
	if err != nil {
		return m, err
	}
	result, err := r.db.ExecContext(ctx, "UPDATE models SET name=$2,category=$3,description=$4,daily_price=$5,image_urls=$6::jsonb WHERE id=$1", m.ID, m.Name, m.Category, m.Description, m.DailyPrice, string(data))
	return m, database.Changed(result, err)
}
func (r *repository) ListUnits(ctx context.Context, id string) ([]entity.Unit, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT id,model_id,state FROM units WHERE model_id=$1 ORDER BY id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	units := []entity.Unit{}
	for rows.Next() {
		var u entity.Unit
		if err = rows.Scan(&u.ID, &u.ModelID, &u.State); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, rows.Err()
}
func (r *repository) AddUnit(ctx context.Context, u entity.Unit) (entity.Unit, error) {
	u.ID = platform.NewID()
	u.State = entity.UnitReady
	_, err := r.db.ExecContext(ctx, "INSERT INTO units(id,model_id,state) VALUES($1,$2,$3)", u.ID, u.ModelID, u.State)
	return u, database.Error(err)
}
func (r *repository) SetUnitState(ctx context.Context, id, state string) (entity.Unit, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Unit{}, err
	}
	defer tx.Rollback()
	var u entity.Unit
	if err = tx.QueryRowContext(ctx, "SELECT id,model_id,state FROM units WHERE id=$1", id).Scan(&u.ID, &u.ModelID, &u.State); err != nil {
		return u, database.Error(err)
	}
	if err = lockModels(ctx, tx, []string{u.ModelID}); err != nil {
		return u, err
	}
	if state != entity.UnitReady {
		var busy bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM allocations a JOIN reservations r USING(order_id)
   WHERE a.unit_id=$1 AND NOT a.cancelled AND NOT a.returned AND r.state NOT IN ('cancelled','completed'))`, id).Scan(&busy)
		if err != nil {
			return u, err
		}
		if busy {
			return u, fmt.Errorf("%w: unit has outstanding reservations", platform.ErrConflict)
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE units SET state=$2 WHERE id=$1", id, state); err != nil {
		return u, database.Error(err)
	}
	u.State = state
	return u, tx.Commit()
}
