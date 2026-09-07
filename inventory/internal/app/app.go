package app

import (
	"context"

	"rent/internal/platform"
	controller "rent/inventory/internal/controller/inventory"
	repository "rent/inventory/internal/repository/inventory"
	usecase "rent/inventory/internal/usecase/inventory"
	pb "rent/pkg/generated"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {

	repo := repository.NewMemory()
	if platform.Env("SEED_DEMO", "false") == "true" {
		if err := repo.Seed(ctx); err != nil {
			return err
		}
	}
	return platform.Serve(ctx, platform.Env("INVENTORY_LISTEN", ":50052"), func(s *grpc.Server) { pb.RegisterInventoryServer(s, controller.New(usecase.New(repo))) })
}
