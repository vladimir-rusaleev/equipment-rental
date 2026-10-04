package entity

import "time"

const (
	RoleCustomer = "customer"
	RoleStaff    = "staff"
	RoleAdmin    = "admin"
)

type User struct {
	ID           string
	Email        string
	Role         string
	PasswordHash string
}

func (u User) IsStaff() bool { return u.Role == RoleStaff || u.Role == RoleAdmin }

type Session struct {
	Token string
	User  User
}
type StoredSession struct {
	UserID    string
	ExpiresAt time.Time
}
