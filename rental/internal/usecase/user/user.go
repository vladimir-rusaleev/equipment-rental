package user

import (
	"context"
	"net/mail"
	"strings"
	"sync"

	"rent/internal/platform"
	"rent/rental/internal/entity"

	"golang.org/x/crypto/bcrypt"
)

type userRepository interface {
	EnsureAdmin(context.Context, string, string) error
	CreateUser(context.Context, entity.User) (entity.User, error)
	UserByEmail(context.Context, string) (entity.User, error)
	User(context.Context, string) (entity.User, error)
	SetRole(context.Context, string, string) (entity.User, error)
	PutSession(context.Context, string, string) error
	SessionUser(context.Context, string) (entity.User, error)
}
type service struct {
	mu   sync.Mutex
	repo userRepository
}

func New(repo userRepository) *service { return &service{repo: repo} }

func (s *service) Register(ctx context.Context, email, password string) (entity.Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || len(password) < 10 || len(password) > 72 {
		return entity.Session{}, platform.ErrInvalid
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.Session{}, err
	}
	u, err := s.repo.CreateUser(ctx, entity.User{Email: email, PasswordHash: string(h), Role: entity.RoleCustomer})
	if err != nil {
		return entity.Session{}, err
	}
	return s.session(ctx, u)
}

func (s *service) Bootstrap(ctx context.Context, email, password string) error {
	if email == "" && password == "" {
		return nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(password) < 10 || len(password) > 72 {
		return platform.ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.EnsureAdmin(ctx, email, string(hash))
}

func (s *service) session(ctx context.Context, u entity.User) (entity.Session, error) {
	t := platform.NewID() + platform.NewID()
	if err := s.repo.PutSession(ctx, t, u.ID); err != nil {
		return entity.Session{}, err
	}
	return entity.Session{Token: t, User: u}, nil
}

func (s *service) Login(ctx context.Context, email, password string) (entity.Session, error) {
	u, err := s.repo.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return entity.Session{}, platform.ErrUnauthenticated
	}
	return s.session(ctx, u)
}

func (s *service) Authenticate(ctx context.Context, token string) (entity.User, error) {
	return s.repo.SessionUser(ctx, token)
}

func (s *service) CurrentUser(ctx context.Context) (entity.User, error) {
	id, _ := platform.Actor(ctx)
	if id == "" {
		return entity.User{}, platform.ErrUnauthenticated
	}
	return s.repo.User(ctx, id)
}

func (s *service) SetRole(ctx context.Context, id, role string) (entity.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, err := s.CurrentUser(ctx)
	if err != nil {
		return entity.User{}, err
	}
	if u.Role != entity.RoleAdmin {
		return entity.User{}, platform.ErrForbidden
	}
	if role != entity.RoleAdmin && role != entity.RoleStaff && role != entity.RoleCustomer {
		return entity.User{}, platform.ErrInvalid
	}
	return s.repo.SetRole(ctx, id, role)
}
