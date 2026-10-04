package order

import (
	"context"
	"log/slog"
	"time"
)

type orderService interface{ ResumePending(context.Context) error }
type worker struct{ orders orderService }

func New(orders orderService) *worker { return &worker{orders: orders} }

func (w *worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.orders.ResumePending(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("resume pending orders", "error", err)
			}
		}
	}
}
