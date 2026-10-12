package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"rent/internal/platform"

	"github.com/lib/pq"
)

// Open connects to one service's database and applies embedded migrations atomically.
func Open(ctx context.Context, dsn string, migrations fs.FS) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		if err = db.PingContext(startup); err == nil {
			break
		}
		select {
		case <-startup.Done():
			db.Close()
			return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
		case <-time.After(500 * time.Millisecond):
		}
	}
	if err = migrate(startup, db, migrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate PostgreSQL: %w", err)
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB, files fs.FS) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(17488)"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
  name text PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now()
 )`); err != nil {
		return err
	}
	names, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return err
	}
	for _, name := range names {
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		checksum := hex.EncodeToString(digest[:])
		var existing string
		err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE name=$1", name).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration %s was modified after applying", name)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(name,checksum) VALUES($1,$2)", name, checksum); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func Error(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return platform.ErrMissing
	}
	var pg *pq.Error
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505", "23P01", "40001", "40P01":
			return fmt.Errorf("%w: database conflict", platform.ErrConflict)
		case "23503":
			return platform.ErrMissing
		case "23514", "22007", "22008", "22003":
			return platform.ErrInvalid
		}
	}
	return err
}

func Changed(result sql.Result, err error) error {
	if err != nil {
		return Error(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return platform.ErrMissing
	}
	return nil
}
