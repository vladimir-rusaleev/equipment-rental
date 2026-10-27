package app

import (
	"context"
	"sync"

	"rent/internal/eventbus"
	"rent/internal/platform"
	database "rent/internal/postgres"
	sender "rent/notifier/internal/adapter/log"
	kafkacontroller "rent/notifier/internal/controller/kafka"
	controller "rent/notifier/internal/controller/notification"
	"rent/notifier/internal/entity"
	repository "rent/notifier/internal/repository/notification"
	postgresrepository "rent/notifier/internal/repository/postgres"
	usecase "rent/notifier/internal/usecase/notification"
	pb "rent/pkg/generated"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	config, err := eventbus.Load()
	if err != nil {
		return err
	}
	storage, err := database.Storage()
	if err != nil {
		return err
	}
	var repo notificationRepository = repository.NewMemory()
	if storage == "postgres" {
		dsn, err := database.DSN("NOTIFIER_DATABASE_URL", "rent_notifier")
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
	s := usecase.New(repo, sender.New())
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	var workers sync.WaitGroup
	workers.Add(1)
	go func() { defer workers.Done(); s.RunWorker(c) }()
	if config.Transport == "kafka" {
		consumer := kafkacontroller.New(config, s)
		defer consumer.Close()
		workers.Add(1)
		go func() { defer workers.Done(); consumer.Run(c) }()
	}
	err = platform.Serve(ctx, platform.Env("NOTIFIER_LISTEN", ":50053"), func(g *grpc.Server) { pb.RegisterNotifierServer(g, controller.New(s)) })
	cancel()
	workers.Wait()
	return err
}

type notificationRepository interface {
	PendingCount(context.Context) (int, error)
	Put(context.Context, entity.Notification) error
	Pending(context.Context) ([]entity.Notification, error)
	Sent(context.Context, string) error
}
