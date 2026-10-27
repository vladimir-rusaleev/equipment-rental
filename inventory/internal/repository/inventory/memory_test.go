package inventory

import (
	"context"
	"errors"
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
