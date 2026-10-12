package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"rent/internal/platform"
	"rent/inventory/internal/entity"

	"github.com/lib/pq"
)

func testRepository(t *testing.T) *repository {
	t.Helper()
	dsn := os.Getenv("INVENTORY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INVENTORY_TEST_DATABASE_URL to a disposable test database")
	}
	repo, err := New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}
func equipment(t *testing.T, repo *repository, quantity int) (entity.Model, []entity.Unit) {
	t.Helper()
	ctx := context.Background()
	m, err := repo.AddModel(ctx, entity.Model{Name: "Test drill", Category: "test", DailyPrice: 100})
	if err != nil {
		t.Fatal(err)
	}
	units := []entity.Unit{}
	for range quantity {
		u, err := repo.AddUnit(ctx, entity.Unit{ModelID: m.ID})
		if err != nil {
			t.Fatal(err)
		}
		units = append(units, u)
	}
	return m, units
}
func request(model string) entity.Item {
	return entity.Item{ModelID: model, Quantity: 1, Start: "2030-11-10", End: "2030-11-12"}
}

func TestConcurrentReservationAcrossConnections(t *testing.T) {
	repo := testRepository(t)
	second := testRepository(t)
	m, _ := equipment(t, repo, 1)
	var wg sync.WaitGroup
	results := make(chan error, 20)
	for n := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := repo
			if n%2 == 0 {
				client = second
			}
			_, err := client.Reserve(context.Background(), platform.NewID(), []entity.Item{request(m.ID)})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, platform.ErrConflict) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("expected 1 reservation, got %d", success)
	}
}
func TestBundleRollbackIdempotencyAndPartialReturn(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	m, units := equipment(t, repo, 2)
	missing, _ := equipment(t, repo, 0)
	if _, err := repo.Reserve(ctx, platform.NewID(), []entity.Item{request(m.ID), request(missing.ID)}); !errors.Is(err, platform.ErrConflict) {
		t.Fatalf("bundle should fail: %v", err)
	}
	calendar, err := repo.Calendar(ctx, m.ID, "2030-11-10", "2030-11-12")
	if err != nil || calendar[0].Available != 2 {
		t.Fatalf("partial bundle leaked: %+v %v", calendar, err)
	}
	item := request(m.ID)
	item.Quantity = 2
	id := platform.NewID()
	original, err := repo.Reserve(ctx, id, []entity.Item{item})
	if err != nil {
		t.Fatal(err)
	}
	m.DailyPrice = 999
	if _, err = repo.UpdateModel(ctx, m); err != nil {
		t.Fatal(err)
	}
	again, err := repo.Reserve(ctx, id, []entity.Item{item})
	if err != nil || again.Items[0].Total != original.Items[0].Total {
		t.Fatal("retry did not preserve price", err)
	}
	item.Quantity = 1
	if _, err = repo.Reserve(ctx, id, []entity.Item{item}); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("different payload accepted", err)
	}
	if _, err = repo.Issue(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SetUnitState(ctx, units[0].ID, entity.UnitMaintenance); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("issued unit put into maintenance", err)
	}
	returned, err := repo.Return(ctx, id, []string{units[0].ID})
	if err != nil || returned.State != entity.ReservationActive {
		t.Fatal("partial return", err)
	}
	returned, err = repo.Return(ctx, id, []string{units[1].ID})
	if err != nil || returned.State != entity.ReservationCompleted {
		t.Fatal("complete return", err)
	}
	reopened := testRepository(t)
	stored, err := reopened.GetReservation(ctx, id)
	if err != nil || stored.State != entity.ReservationCompleted || len(stored.Items[0].ReturnedUnitIDs) != 2 {
		t.Fatal("reopen lost data", err)
	}
}
func TestDatabaseExclusionConstraint(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	m, units := equipment(t, repo, 1)
	if _, err := repo.Reserve(ctx, platform.NewID(), []entity.Item{request(m.ID)}); err != nil {
		t.Fatal(err)
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	id := platform.NewID()
	if _, err = tx.ExecContext(ctx, "INSERT INTO reservations VALUES($1,'reserved','[]')", id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO reservation_items VALUES($1,0,$2,'Test',1,'2030-11-10','2030-11-12',100,200)`, id, m.ID); err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO allocations(order_id,line_no,unit_id,period) VALUES($1,0,$2,daterange('2030-11-10','2030-11-12','[)'))", id, units[0].ID)
	var pg *pq.Error
	if !errors.As(err, &pg) || pg.Code != "23P01" {
		t.Fatalf("database accepted overlap: %v", err)
	}
}
func TestSeedDoesNotDuplicateOrResetPrices(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	if err := repo.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	model, err := repo.Get(ctx, "demo-bosch-gbh226")
	if err != nil {
		t.Fatal(err)
	}
	model.DailyPrice = 12345
	if _, err = repo.UpdateModel(ctx, model); err != nil {
		t.Fatal(err)
	}
	if err = repo.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	model, err = repo.Get(ctx, model.ID)
	if err != nil || model.DailyPrice != 12345 {
		t.Fatal("seed reset edited price", err)
	}
	units, err := repo.ListUnits(ctx, model.ID)
	if err != nil || len(units) != 3 {
		t.Fatal("seed duplicated units", err)
	}
}
