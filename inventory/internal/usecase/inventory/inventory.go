package inventory

import (
	"context"
	"strings"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
)

type repository interface {
	List(context.Context, string, string, string, string) ([]entity.Model, error)
	Get(context.Context, string) (entity.Model, error)
	AddModel(context.Context, entity.Model) (entity.Model, error)
	UpdateModel(context.Context, entity.Model) (entity.Model, error)
	ListUnits(context.Context, string) ([]entity.Unit, error)
	AddUnit(context.Context, entity.Unit) (entity.Unit, error)
	SetUnitState(context.Context, string, string) (entity.Unit, error)
	Calendar(context.Context, string, string, string) ([]entity.Day, error)
	Reserve(context.Context, string, []entity.Item) (entity.Reservation, error)
	GetReservation(context.Context, string) (entity.Reservation, error)
	Cancel(context.Context, string) (entity.Reservation, error)
	Issue(context.Context, string) (entity.Reservation, error)
	Return(context.Context, string, []string) (entity.Reservation, error)
}
type service struct{ repo repository }

func New(r repository) *service { return &service{repo: r} }

func (s *service) List(ctx context.Context, q, c, a, b string) ([]entity.Model, error) {
	if a != "" || b != "" {
		if _, _, err := entity.Period(a, b); err != nil {
			return nil, err
		}
	}
	return s.repo.List(ctx, q, c, a, b)
}

func (s *service) Get(ctx context.Context, id string) (entity.Model, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) AddModel(ctx context.Context, m entity.Model) (entity.Model, error) {
	if err := authorize(ctx, true); err != nil {
		return entity.Model{}, err
	}
	if strings.TrimSpace(m.Name) == "" || m.Category == "" || m.DailyPrice < 1 || m.DailyPrice > 100000000 {
		return m, platform.ErrInvalid
	}
	return s.repo.AddModel(ctx, m)
}

func (s *service) AddUnit(ctx context.Context, u entity.Unit) (entity.Unit, error) {
	if err := authorize(ctx, false); err != nil {
		return entity.Unit{}, err
	}
	return s.repo.AddUnit(ctx, u)
}

func (s *service) SetUnitState(ctx context.Context, id, state string) (entity.Unit, error) {
	if err := authorize(ctx, false); err != nil {
		return entity.Unit{}, err
	}
	if state != entity.UnitReady && state != entity.UnitMaintenance && state != entity.UnitRetired {
		return entity.Unit{}, platform.ErrInvalid
	}
	return s.repo.SetUnitState(ctx, id, state)
}

func (s *service) Calendar(ctx context.Context, id, a, b string) ([]entity.Day, error) {
	if _, _, err := entity.Period(a, b); err != nil {
		return nil, err
	}
	return s.repo.Calendar(ctx, id, a, b)
}

func (s *service) Reserve(ctx context.Context, id string, items []entity.Item) (entity.Reservation, error) {
	if id == "" {
		return entity.Reservation{}, platform.ErrInvalid
	}
	if err := entity.Validate(items); err != nil {
		return entity.Reservation{}, err
	}
	return s.repo.Reserve(ctx, id, items)
}

func (s *service) GetReservation(ctx context.Context, id string) (entity.Reservation, error) {
	return s.repo.GetReservation(ctx, id)
}

func (s *service) Cancel(ctx context.Context, id string) (entity.Reservation, error) {
	return s.repo.Cancel(ctx, id)
}

func (s *service) Issue(ctx context.Context, id string) (entity.Reservation, error) {
	return s.repo.Issue(ctx, id)
}

func (s *service) Return(ctx context.Context, id string, ids []string) (entity.Reservation, error) {
	return s.repo.Return(ctx, id, ids)
}

func authorize(ctx context.Context, admin bool) error {
	_, role := platform.Actor(ctx)
	if role == entity.RoleAdmin || !admin && role == entity.RoleStaff {
		return nil
	}
	return platform.ErrForbidden
}

func (s *service) UpdateModel(ctx context.Context, m entity.Model) (entity.Model, error) {
	if err := authorize(ctx, true); err != nil {
		return m, err
	}
	if strings.TrimSpace(m.Name) == "" || m.Category == "" || m.DailyPrice < 1 || m.DailyPrice > 100000000 {
		return m, platform.ErrInvalid
	}
	return s.repo.UpdateModel(ctx, m)
}

func (s *service) ListUnits(ctx context.Context, id string) ([]entity.Unit, error) {
	if err := authorize(ctx, false); err != nil {
		return nil, err
	}
	return s.repo.ListUnits(ctx, id)
}
