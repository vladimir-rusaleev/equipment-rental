package notification

import (
	"context"
	"time"

	"rent/internal/platform"
	"rent/notifier/internal/entity"
)

type repository interface {
	Put(context.Context, entity.Notification) error
	Pending(context.Context) ([]entity.Notification, error)
	Sent(context.Context, string) error
}
type sender interface {
	Send(context.Context, entity.Notification) error
}
type service struct {
	repo   repository
	sender sender
}

func New(r repository, s sender) *service { return &service{repo: r, sender: s} }

func (s *service) Notify(ctx context.Context, n entity.Notification) error {
	if n.ID == "" || n.UserID == "" || n.OrderID == "" || n.State == "" {
		return platform.ErrInvalid
	}
	return s.repo.Put(ctx, n)
}

func (s *service) RunWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			items, err := s.repo.Pending(ctx)
			if err != nil {
				continue
			}
			for _, n := range items {
				c, cancel := context.WithTimeout(ctx, 3*time.Second)
				err = s.sender.Send(c, n)
				cancel()
				if err == nil {
					_ = s.repo.Sent(ctx, n.ID)
				}
			}
		}
	}
}
