package app

import (
	"context"

	"rent/internal/platform"
	database "rent/internal/postgres"
	controller "rent/inventory/internal/controller/inventory"
	"rent/inventory/internal/entity"
	repository "rent/inventory/internal/repository/inventory"
	postgresrepository "rent/inventory/internal/repository/postgres"
	usecase "rent/inventory/internal/usecase/inventory"
	pb "rent/pkg/generated"

	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {

	storage, err := database.Storage()
	if err != nil {
		return err
	}
	var repo inventoryRepository = repository.NewMemory()
	if storage == "postgres" {
		dsn, err := database.DSN("INVENTORY_DATABASE_URL", "rent_inventory")
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
	if platform.Env("SEED_DEMO", "false") == "true" {
		if err = repo.Seed(ctx); err != nil {
			return err
		}
	}
	return platform.Serve(ctx, platform.Env("INVENTORY_LISTEN", ":50052"), func(s *grpc.Server) { pb.RegisterInventoryServer(s, controller.New(usecase.New(repo))) })
}

type inventoryRepository interface {
	Seed(context.Context) error
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
