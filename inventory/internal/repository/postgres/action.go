package postgres

import (
	"context"
	"fmt"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/inventory/internal/entity"

	"github.com/lib/pq"
)

func (r *repository) Cancel(ctx context.Context, id string) (entity.Reservation, error) {
	return r.change(ctx, id, "cancel", nil)
}
func (r *repository) Issue(ctx context.Context, id string) (entity.Reservation, error) {
	return r.change(ctx, id, "issue", nil)
}
func (r *repository) Return(ctx context.Context, id string, units []string) (entity.Reservation, error) {
	return r.change(ctx, id, "return", units)
}

func (r *repository) change(ctx context.Context, id, action string, units []string) (entity.Reservation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Reservation{}, err
	}
	defer tx.Rollback()
	if err = lockReservation(ctx, tx, id); err != nil {
		return entity.Reservation{}, err
	}
	result, err := getReservation(ctx, tx, id)
	if err != nil {
		return result, err
	}
	models := []string{}
	for _, item := range result.Items {
		models = append(models, item.Item.ModelID)
	}
	if err = lockModels(ctx, tx, models); err != nil {
		return result, err
	}
	switch action {
	case "cancel":
		if result.State != entity.ReservationReserved && result.State != entity.ReservationCancelled {
			return result, platform.ErrConflict
		}
		if _, err = tx.ExecContext(ctx, "UPDATE allocations SET cancelled=true WHERE order_id=$1", id); err != nil {
			return result, database.Error(err)
		}
		result.State = entity.ReservationCancelled
	case "issue":
		if result.State == entity.ReservationActive {
			return result, nil
		}
		if result.State != entity.ReservationReserved {
			return result, platform.ErrConflict
		}
		var busy bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM allocations a JOIN units u ON u.id=a.unit_id
   WHERE a.order_id=$1 AND (u.state<>'ready' OR EXISTS(
    SELECT 1 FROM allocations other WHERE other.unit_id=a.unit_id AND other.order_id<>$1
     AND other.issued AND NOT other.returned AND NOT other.cancelled)))`, id).Scan(&busy)
		if err != nil {
			return result, err
		}
		if busy {
			return result, fmt.Errorf("%w: unit not physically returned or not ready", platform.ErrConflict)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE allocations SET issued=true WHERE order_id=$1", id); err != nil {
			return result, database.Error(err)
		}
		result.State = entity.ReservationActive
	case "return":
		if result.State != entity.ReservationActive && result.State != entity.ReservationCompleted {
			return result, platform.ErrConflict
		}
		if len(units) == 0 {
			return result, platform.ErrInvalid
		}
		known := map[string]bool{}
		for _, item := range result.Items {
			for _, unit := range item.UnitIDs {
				known[unit] = true
			}
		}
		for _, unit := range units {
			if !known[unit] {
				return result, platform.ErrInvalid
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE allocations SET returned=true WHERE order_id=$1 AND unit_id=ANY($2)", id, pq.Array(units)); err != nil {
			return result, database.Error(err)
		}
		var outstanding bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM allocations WHERE order_id=$1 AND NOT returned)", id).Scan(&outstanding); err != nil {
			return result, err
		}
		if !outstanding {
			result.State = entity.ReservationCompleted
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE reservations SET state=$2 WHERE order_id=$1", id, result.State); err != nil {
		return result, database.Error(err)
	}
	result, err = getReservation(ctx, tx, id)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
