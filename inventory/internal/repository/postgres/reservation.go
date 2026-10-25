package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/inventory/internal/entity"

	"github.com/lib/pq"
)

func lockModels(ctx context.Context, tx *sql.Tx, ids []string) error {
	// Every operation acquires model locks in the same order.
	sort.Strings(ids)
	rows, err := tx.QueryContext(ctx, "SELECT id FROM models WHERE id=ANY($1) ORDER BY id FOR UPDATE", pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return err
		}
		found[id] = true
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if !found[id] {
			return platform.ErrMissing
		}
	}
	return nil
}
func lockReservation(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", id)
	return err
}

func (r *repository) Reserve(ctx context.Context, id string, items []entity.Item) (entity.Reservation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Reservation{}, err
	}
	defer tx.Rollback()
	if err = lockReservation(ctx, tx, id); err != nil {
		return entity.Reservation{}, err
	}
	var data []byte
	err = tx.QueryRowContext(ctx, "SELECT request FROM reservations WHERE order_id=$1", id).Scan(&data)
	if err == nil {
		var previous []entity.Item
		if err = json.Unmarshal(data, &previous); err != nil {
			return entity.Reservation{}, err
		}
		if !reflect.DeepEqual(previous, items) {
			return entity.Reservation{}, platform.ErrConflict
		}
		result, err := getReservation(ctx, tx, id)
		return result, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return entity.Reservation{}, err
	}
	ids := []string{}
	for _, item := range items {
		ids = append(ids, item.ModelID)
	}
	if err = lockModels(ctx, tx, ids); err != nil {
		return entity.Reservation{}, err
	}
	data, err = json.Marshal(items)
	if err != nil {
		return entity.Reservation{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO reservations(order_id,state,request) VALUES($1,'reserved',$2::jsonb)", id, string(data)); err != nil {
		return entity.Reservation{}, database.Error(err)
	}
	for line, item := range items {
		model, err := scanModel(tx.QueryRowContext(ctx, "SELECT "+modelColumns+" FROM models WHERE id=$1", item.ModelID))
		if err != nil {
			return entity.Reservation{}, err
		}
		free, err := available(ctx, tx, item.ModelID, item.Start, item.End)
		if err != nil {
			return entity.Reservation{}, err
		}
		if len(free) < int(item.Quantity) {
			return entity.Reservation{}, fmt.Errorf("%w: insufficient equipment for %s", platform.ErrConflict, item.ModelID)
		}
		start, end, err := entity.Period(item.Start, item.End)
		if err != nil {
			return entity.Reservation{}, err
		}
		total := int64(end.Sub(start)/(24*time.Hour)) * int64(item.Quantity) * model.DailyPrice
		_, err = tx.ExecContext(ctx, `INSERT INTO reservation_items(order_id,line_no,model_id,name,quantity,starts_on,ends_on,daily_price,total)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, line, item.ModelID, model.Name, item.Quantity, item.Start, item.End, model.DailyPrice, total)
		if err != nil {
			return entity.Reservation{}, database.Error(err)
		}
		for _, unit := range free[:int(item.Quantity)] {
			_, err = tx.ExecContext(ctx, "INSERT INTO allocations(order_id,line_no,unit_id,period) VALUES($1,$2,$3,daterange($4::date,$5::date,'[)'))", id, line, unit, item.Start, item.End)
			if err != nil {
				return entity.Reservation{}, database.Error(err)
			}
		}
	}
	result, err := getReservation(ctx, tx, id)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}

func (r *repository) GetReservation(ctx context.Context, id string) (entity.Reservation, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return entity.Reservation{}, err
	}
	defer tx.Rollback()
	result, err := getReservation(ctx, tx, id)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}

func getReservation(ctx context.Context, q queryer, id string) (entity.Reservation, error) {
	result := entity.Reservation{OrderID: id, Items: []entity.Quote{}}
	if err := q.QueryRowContext(ctx, "SELECT state FROM reservations WHERE order_id=$1", id).Scan(&result.State); err != nil {
		return result, database.Error(err)
	}
	rows, err := q.QueryContext(ctx, `SELECT model_id,name,quantity,starts_on,ends_on,daily_price,total
  FROM reservation_items WHERE order_id=$1 ORDER BY line_no`, id)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item entity.Quote
		var start, end time.Time
		if err = rows.Scan(&item.Item.ModelID, &item.Name, &item.Item.Quantity, &start, &end, &item.DailyPrice, &item.Total); err != nil {
			rows.Close()
			return result, err
		}
		item.Item.Start = start.Format(time.DateOnly)
		item.Item.End = end.Format(time.DateOnly)
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = q.QueryContext(ctx, "SELECT line_no,unit_id,returned FROM allocations WHERE order_id=$1 ORDER BY line_no,unit_id", id)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var line int
		var unit string
		var returned bool
		if err = rows.Scan(&line, &unit, &returned); err != nil {
			return result, err
		}
		result.Items[line].UnitIDs = append(result.Items[line].UnitIDs, unit)
		if returned {
			result.Items[line].ReturnedUnitIDs = append(result.Items[line].ReturnedUnitIDs, unit)
		}
	}
	return result, rows.Err()
}
