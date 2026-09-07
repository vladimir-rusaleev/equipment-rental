package inventory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"rent/internal/platform"
	"rent/inventory/internal/entity"
)

func setup(t *testing.T) (*memory, entity.Model, entity.Unit) {
	t.Helper()
	m := NewMemory()
	model, err := m.AddModel(context.Background(), entity.Model{Name: "Drill", Category: "drills", DailyPrice: 100})
	if err != nil {
		t.Fatal(err)
	}
	u, err := m.AddUnit(context.Background(), entity.Unit{ModelID: model.ID})
	if err != nil {
		t.Fatal(err)
	}
	return m, model, u
}

func item(id string) entity.Item {
	return entity.Item{ModelID: id, Quantity: 1, Start: "2026-11-10", End: "2026-11-12"}
}

func TestConcurrentReservations(t *testing.T) {
	m, model, _ := setup(t)
	var wg sync.WaitGroup
	results := make(chan error, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Reserve(context.Background(), platform.NewID(), []entity.Item{item(model.ID)})
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
		t.Fatalf("successful reservations: %d", success)
	}
}

func TestAtomicBundleAndIdempotency(t *testing.T) {
	m, model, _ := setup(t)
	ctx := context.Background()
	_, err := m.Reserve(ctx, "failed", []entity.Item{item(model.ID), item(model.ID)})
	if !errors.Is(err, platform.ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
	r, err := m.Reserve(ctx, "ok", []entity.Item{item(model.ID)})
	if err != nil {
		t.Fatal(err)
	}
	again, err := m.Reserve(ctx, "ok", []entity.Item{item(model.ID)})
	if err != nil || again.Items[0].UnitIDs[0] != r.Items[0].UnitIDs[0] {
		t.Fatal("idempotency failed")
	}
	changed := item(model.ID)
	changed.Quantity = 2
	if _, err = m.Reserve(ctx, "ok", []entity.Item{changed}); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("payload mismatch accepted")
	}
	r.Items[0].UnitIDs[0] = "mutated"
	stored, _ := m.GetReservation(ctx, "ok")
	if stored.Items[0].UnitIDs[0] == "mutated" {
		t.Fatal("repository leaked mutable state")
	}
}

func TestAdjacentPeriodsCancellationAndReturn(t *testing.T) {
	m, model, u := setup(t)
	ctx := context.Background()
	a := item(model.ID)
	if _, err := m.Reserve(ctx, "a", []entity.Item{a}); err != nil {
		t.Fatal(err)
	}
	b := a
	b.Start = a.End
	b.End = "2026-11-14"
	if _, err := m.Reserve(ctx, "b", []entity.Item{b}); err != nil {
		t.Fatalf("adjacent periods must not conflict: %v", err)
	}
	if _, err := m.Cancel(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Issue(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Cancel(ctx, "a"); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("issued reservation cancelled")
	}
	if _, err := m.SetUnitState(ctx, u.ID, "maintenance"); !errors.Is(err, platform.ErrConflict) {
		t.Fatal("reserved unit put in maintenance")
	}
	if _, err := m.Return(ctx, "a", []string{"not-a-unit"}); !errors.Is(err, platform.ErrInvalid) {
		t.Fatal("unknown return accepted")
	}
	r, err := m.Return(ctx, "a", []string{u.ID})
	if err != nil || r.State != "completed" {
		t.Fatalf("return: %+v %v", r, err)
	}
	r, err = m.Return(ctx, "a", []string{u.ID})
	if err != nil || len(r.Items[0].ReturnedUnitIDs) != 1 {
		t.Fatal("duplicate return not idempotent")
	}
}

func TestDifferentPeriodsWithinBundle(t *testing.T) {
	m, model, _ := setup(t)
	a := item(model.ID)
	b := a
	b.Start = a.End
	b.End = "2026-11-15"
	r, err := m.Reserve(context.Background(), "bundle", []entity.Item{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if r.Items[0].UnitIDs[0] != r.Items[1].UnitIDs[0] {
		t.Fatal("same unit should be reusable in adjacent intervals")
	}
	if r.Items[0].Total != 200 || r.Items[1].Total != 300 {
		t.Fatal("wrong daily pricing")
	}
}

func TestPriceSnapshotAndImageIsolation(t *testing.T) {
	m, model, _ := setup(t)
	ctx := context.Background()
	model.ImageURLs = []string{"https://example.com/drill.png"}
	updated, err := m.UpdateModel(ctx, model)
	if err != nil {
		t.Fatal(err)
	}
	updated.ImageURLs[0] = "changed"
	stored, _ := m.Get(ctx, model.ID)
	if stored.ImageURLs[0] != "https://example.com/drill.png" {
		t.Fatal("model leaked mutable state")
	}
	reserved, err := m.Reserve(ctx, "snapshot", []entity.Item{item(model.ID)})
	if err != nil {
		t.Fatal(err)
	}
	model.DailyPrice = 500
	_, _ = m.UpdateModel(ctx, model)
	again, err := m.Reserve(ctx, "snapshot", []entity.Item{item(model.ID)})
	if err != nil || again.Items[0].Total != reserved.Items[0].Total {
		t.Fatal("existing price changed")
	}
}
