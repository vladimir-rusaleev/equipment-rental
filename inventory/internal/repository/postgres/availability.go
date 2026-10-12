package postgres

import (
	"context"
	"time"

	database "rent/internal/postgres"
	"rent/inventory/internal/entity"
)

func available(ctx context.Context, q queryer, model, start, end string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT u.id FROM units u WHERE u.model_id=$1 AND u.state='ready'
 AND NOT EXISTS(SELECT 1 FROM allocations a WHERE a.unit_id=u.id AND NOT a.cancelled
  AND (a.period && daterange($2::date,$3::date,'[)')
       OR (a.issued AND NOT a.returned AND upper(a.period)<=(now() AT TIME ZONE 'UTC')::date)))
 ORDER BY u.id`, model, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *repository) Calendar(ctx context.Context, id, start, end string) ([]entity.Day, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT d::date,
 (SELECT count(*) FROM units u WHERE u.model_id=$1 AND u.state='ready'
  AND NOT EXISTS(SELECT 1 FROM allocations a WHERE a.unit_id=u.id AND NOT a.cancelled
   AND (a.period && daterange(d::date,d::date+1,'[)')
        OR(a.issued AND NOT a.returned AND upper(a.period)<=(now() AT TIME ZONE 'UTC')::date))))
 FROM generate_series($2::date, $3::date-1, interval '1 day') d ORDER BY d`, id, start, end)
	if err != nil {
		return nil, database.Error(err)
	}
	defer rows.Close()
	days := []entity.Day{}
	for rows.Next() {
		var date time.Time
		var day entity.Day
		if err = rows.Scan(&date, &day.Available); err != nil {
			return nil, err
		}
		day.Date = date.Format(time.DateOnly)
		days = append(days, day)
	}
	return days, rows.Err()
}
