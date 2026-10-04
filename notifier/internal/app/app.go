package app

import (
	"context"

	"rent/internal/platform"
	sender "rent/notifier/internal/adapter/log"
	controller "rent/notifier/internal/controller/notification"
	repository "rent/notifier/internal/repository/notification"
	usecase "rent/notifier/internal/usecase/notification"
	pb "rent/pkg/generated"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	repo := repository.NewMemory()
	s := usecase.New(repo, sender.New())
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); s.RunWorker(c) }()
	err := platform.Serve(ctx, platform.Env("NOTIFIER_LISTEN", ":50053"), func(g *grpc.Server) { pb.RegisterNotifierServer(g, controller.New(s)) })
	cancel()
	<-done
	return err
}
