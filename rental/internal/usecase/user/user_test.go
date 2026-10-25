package user_test

import (
	"context"
	"errors"
	"testing"

	"rent/internal/platform"
	rentalrepository "rent/rental/internal/repository/rental"
	userusecase "rent/rental/internal/usecase/user"
)

func TestRolesAndSessions(t *testing.T) {
	repo := rentalrepository.NewMemory()
	s := userusecase.New(repo)
	ctx := context.Background()
	ss, err := s.Register(ctx, "user@example.com", "strong-password")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Authenticate(ctx, ss.Token)
	if err != nil || u.Role != "customer" {
		t.Fatal("session failed")
	}
	if _, err = s.SetRole(platform.WithActor(ctx, u.ID, ""), u.ID, "admin"); !errors.Is(err, platform.ErrForbidden) {
		t.Fatal("self elevation allowed")
	}
	if _, err = s.Login(ctx, "user@example.com", "wrong-password"); !errors.Is(err, platform.ErrUnauthenticated) {
		t.Fatal("wrong password accepted")
	}
	if _, err = s.Authenticate(ctx, "invalid"); !errors.Is(err, platform.ErrUnauthenticated) {
		t.Fatal("invalid token accepted")
	}
}
