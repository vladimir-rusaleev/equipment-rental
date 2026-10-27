package app

import (
	"context"
	"sync"

	"rent/internal/eventbus"
	"rent/internal/platform"
	database "rent/internal/postgres"
	pb "rent/pkg/generated"
	inventory "rent/rental/internal/adapter/inventory/grpc"
	notifier "rent/rental/internal/adapter/notifier/grpc"
	kafkanotifier "rent/rental/internal/adapter/notifier/kafka"
	controller "rent/rental/internal/controller/rental"
	"rent/rental/internal/entity"
	postgresrepository "rent/rental/internal/repository/postgres"
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

	config, err := eventbus.Load()
	if err != nil {
		return err
	}
	var notifications notificationPublisher
	if config.Transport == "kafka" {
		publisher := kafkanotifier.New(config)
		defer publisher.Close()
		notifications = publisher
	} else {
		conn, err := platform.Dial(platform.Env("NOTIFIER_ADDR", "localhost:50053"))
		if err != nil {
			return err
		}
		defer conn.Close()
		notifications = notifier.New(conn)
	}
	storage, err := database.Storage()
	if err != nil {
		return err
	}
	var repo rentalRepository = repository.NewMemory()
	if storage == "postgres" {
		dsn, err := database.DSN("RENTAL_DATABASE_URL", "rent_rental")
		if err != nil {
			return err
		}
		persistent, err := postgresrepository.New(ctx, dsn)
		if err != nil {
			return err
		}
		defer persistent.Close()
		repo = persistent
	}
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
		notificationworker.New(repo, notifications).Run(workerCtx)
	}()
	err = platform.Serve(ctx, platform.Env("RENTAL_LISTEN", ":50051"), func(server *grpc.Server) { pb.RegisterRentalServer(server, controller.New(users, carts, orders)) })
	cancel()
	workers.Wait()
	return err
}

type rentalRepository interface {
	PendingCount(context.Context) (int, error)
	EnsureAdmin(context.Context, string, string) error
	CreateUser(context.Context, entity.User) (entity.User, error)
	UserByEmail(context.Context, string) (entity.User, error)
	User(context.Context, string) (entity.User, error)
	SetRole(context.Context, string, string) (entity.User, error)
	PutSession(context.Context, string, string) error
	SessionUser(context.Context, string) (entity.User, error)
	GetCart(context.Context, string) (entity.Cart, error)
	SetCart(context.Context, string, []entity.Item) (entity.Cart, error)
	FindRequest(context.Context, string, string) (entity.Order, error)
	CreateOrder(context.Context, entity.Order) error
	GetOrder(context.Context, string) (entity.Order, error)
	SaveOrder(context.Context, entity.Order, *entity.Notification, bool) error
	ListOrders(context.Context) ([]entity.Order, error)
	Notifications(context.Context) ([]entity.Notification, error)
	AckNotification(context.Context, string) error
}

type notificationPublisher interface {
	Notify(context.Context, entity.Notification) error
}
