package postgres

import (
	"context"
	"database/sql"
	"embed"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/notifier/internal/entity"
)

//go:embed migrations/*.sql
var migrations embed.FS

type repository struct{ db *sql.DB }

func New(ctx context.Context, dsn string) (*repository, error) {
	db, err := database.Open(ctx, dsn, migrations)
	if err != nil {
		return nil, err
	}
	return &repository{db: db}, nil
}
func (r *repository) Close() error { return r.db.Close() }
func (r *repository) Put(ctx context.Context, n entity.Notification) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO notifications(id,user_id,order_id,state) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING", n.ID, n.UserID, n.OrderID, n.State)
	if err != nil {
		return database.Error(err)
	}
	var existing entity.Notification
	err = r.db.QueryRowContext(ctx, "SELECT id,user_id,order_id,state FROM notifications WHERE id=$1", n.ID).Scan(&existing.ID, &existing.UserID, &existing.OrderID, &existing.State)
	if err != nil {
		return err
	}
	if existing != n {
		return platform.ErrConflict
	}
	return nil
}
func (r *repository) Pending(ctx context.Context) ([]entity.Notification, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,user_id,order_id,state FROM notifications WHERE NOT sent ORDER BY created_at,id LIMIT 100")
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
func (r *repository) Sent(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE notifications SET sent=true,sent_at=now() WHERE id=$1", id)
	return database.Changed(result, err)
}

func (r *repository) PendingCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM notifications WHERE NOT sent").Scan(&count)
	return count, err
}
