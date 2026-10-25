package inventory

import (
	"context"
	"rent/inventory/internal/entity"
)

func (m *memory) Seed(ctx context.Context) error {
	for _, model := range []entity.Model{
		{Name: "Bosch GBH 2-26", Category: "rotary-hammers", Description: "Перфоратор, 800 Вт", DailyPrice: 150000},
		{Name: "Makita 4329", Category: "jigsaws", Description: "Электрический лобзик", DailyPrice: 80000},
	} {
		created, err := m.AddModel(ctx, model)
		if err != nil {
			return err
		}
		for range 3 {
			if _, err = m.AddUnit(ctx, entity.Unit{ModelID: created.ID}); err != nil {
				return err
			}
		}
	}
	return nil
}
