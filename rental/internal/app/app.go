package app

import (
	"context"
	"sync"

	"rent/internal/platform"
	pb "rent/pkg/generated"
	inventory "rent/rental/internal/adapter/inventory/grpc"
	notifier "rent/rental/internal/adapter/notifier/grpc"
	controller "rent/rental/internal/controller/rental"
	repository "rent/rental/internal/repository/rental"
	cartusecase "rent/rental/internal/usecase/cart"
	orderusecase "rent/rental/internal/usecase/order"
	userusecase "rent/rental/internal/usecase/user"
	notificationworker "rent/rental/internal/worker/notification"
	orderworker "rent/rental/internal/worker/order"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	inventoryConn, err := platform.Dial(platform.Env("INVENTORY_ADDR", "localhost:50052"))
	if err != nil {
		return err
	}
	defer inventoryConn.Close()
	notifierConn, err := platform.Dial(platform.Env("NOTIFIER_ADDR", "localhost:50053"))
	if err != nil {
		return err
	}
	defer notifierConn.Close()
	repo := repository.NewMemory()
	users := userusecase.New(repo)
	carts := cartusecase.New(repo, users)
	orders := orderusecase.New(repo, repo, users, inventory.New(inventoryConn))
	if err = users.Bootstrap(ctx, platform.Env("ADMIN_EMAIL", ""), platform.Env("ADMIN_PASSWORD", "")); err != nil {
		return err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var workers sync.WaitGroup
	workers.Add(2)
	go func() { defer workers.Done(); orderworker.New(orders).Run(workerCtx) }()
	go func() {
		defer workers.Done()
		notificationworker.New(repo, notifier.New(notifierConn)).Run(workerCtx)
	}()
	err = platform.Serve(ctx, platform.Env("RENTAL_LISTEN", ":50051"), func(server *grpc.Server) { pb.RegisterRentalServer(server, controller.New(users, carts, orders)) })
	cancel()
	workers.Wait()
	return err
}
