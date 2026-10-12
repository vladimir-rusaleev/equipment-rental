package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"rent/internal/platform"
	database "rent/internal/postgres"
	"rent/rental/internal/entity"
)

const userColumns = "id,email,role,password_hash"

type scanner interface{ Scan(...any) error }

func scanUser(row scanner) (entity.User, error) {
	var user entity.User
	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.PasswordHash)
	return user, database.Error(err)
}

func (r *repository) CreateUser(ctx context.Context, u entity.User) (entity.User, error) {
	u.ID = platform.NewID()
	_, err := r.db.ExecContext(ctx, "INSERT INTO users(id,email,role,password_hash) VALUES($1,$2,$3,$4)", u.ID, u.Email, u.Role, u.PasswordHash)
	return u, database.Error(err)
}
func (r *repository) UserByEmail(ctx context.Context, email string) (entity.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE email=$1", email))
}
func (r *repository) User(ctx context.Context, id string) (entity.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1", id))
}

// EnsureAdmin provisions once; restart never resets a password or elevates an existing customer.
func (r *repository) EnsureAdmin(ctx context.Context, email, hash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(21102)"); err != nil {
		return err
	}
	var role string
	err = tx.QueryRowContext(ctx, "SELECT role FROM users WHERE email=$1", email).Scan(&role)
	switch {
	case err == nil:
		if role != entity.RoleAdmin {
			return platform.ErrConflict
		}
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, "INSERT INTO users(id,email,role,password_hash) VALUES($1,$2,'admin',$3)", platform.NewID(), email, hash)
		if err != nil {
			return database.Error(err)
		}
	default:
		return err
	}
	return tx.Commit()
}

func (r *repository) SetRole(ctx context.Context, id, role string) (entity.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.User{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(21102)"); err != nil {
		return entity.User{}, err
	}
	user, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1 FOR UPDATE", id))
	if err != nil {
		return user, err
	}
	if user.Role == entity.RoleAdmin && role != entity.RoleAdmin {
		var count int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE role='admin'").Scan(&count); err != nil {
			return user, err
		}
		if count == 1 {
			return user, platform.ErrConflict
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE users SET role=$2 WHERE id=$1", id, role); err != nil {
		return user, database.Error(err)
	}
	user.Role = role
	return user, tx.Commit()
}
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (r *repository) PutSession(ctx context.Context, token, id string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", tokenHash(token), id, time.Now().UTC().Add(24*time.Hour))
	return database.Error(err)
}
func (r *repository) SessionUser(ctx context.Context, token string) (entity.User, error) {
	var id string
	err := r.db.QueryRowContext(ctx, "SELECT user_id FROM sessions WHERE token_hash=$1 AND expires_at>now()", tokenHash(token)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.User{}, platform.ErrUnauthenticated
	}
	if err != nil {
		return entity.User{}, err
	}
	return r.User(ctx, id)
}
