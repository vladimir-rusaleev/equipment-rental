package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"rent/frontend"
	"rent/gateway/internal/adapter"
	"rent/gateway/internal/controller/httpapi"
	"rent/internal/platform"
	pb "rent/pkg/generated"
)

func Run(ctx context.Context) error {
	if len(platform.Env("SERVICE_KEY", "")) < 16 {
		return errors.New("SERVICE_KEY must contain at least 16 characters")
	}
	rc, err := platform.Dial(platform.Env("RENTAL_ADDR", "localhost:50051"))
	if err != nil {
		return err
	}
	defer rc.Close()
	ic, err := platform.Dial(platform.Env("INVENTORY_ADDR", "localhost:50052"))
	if err != nil {
		return err
	}
	defer ic.Close()
	mux := http.NewServeMux()
	api := httpapi.New(adapter.New(pb.NewRentalClient(rc), pb.NewInventoryClient(ic)))
	mux.Handle("/api/", api)
	mux.Handle("/healthz", api)
	mux.Handle("/", frontend.Handler())
	s := &http.Server{Addr: platform.Env("GATEWAY_LISTEN", ":8080"), Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Shutdown(c)
		case <-done:
		}
	}()
	err = s.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
