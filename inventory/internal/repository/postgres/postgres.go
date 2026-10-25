package postgres

import (
	"context"
	"database/sql"
	"embed"

	database "rent/internal/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

type repository struct{ db *sql.DB }
type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type scanner interface{ Scan(...any) error }

func New(ctx context.Context, dsn string) (*repository, error) {
	db, err := database.Open(ctx, dsn, migrations)
	if err != nil {
		return nil, err
	}
	return &repository{db: db}, nil
}
func (r *repository) Close() error { return r.db.Close() }
