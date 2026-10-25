package entity

import (
	"fmt"
	"time"

	"rent/internal/platform"
)

type Model struct {
	ImageURLs                       []string
	ID, Name, Category, Description string
	DailyPrice                      int64
	Available                       int32
}
type Unit struct{ ID, ModelID, State string }
type Item struct {
	ModelID    string
	Quantity   int32
	Start, End string
}
type Quote struct {
	Item                     Item
	Name                     string
	DailyPrice, Total        int64
	UnitIDs, ReturnedUnitIDs []string
}
type Reservation struct {
	OrderID, State string
	Items          []Quote
}
type Day struct {
	Date      string
	Available int32
}

func Period(start, end string) (time.Time, time.Time, error) {
	a, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return a, a, fmt.Errorf("%w: start must be YYYY-MM-DD", platform.ErrInvalid)
	}
	b, err := time.Parse(time.DateOnly, end)
	if err != nil || !b.After(a) || b.Sub(a) > 366*24*time.Hour {
		return a, b, fmt.Errorf("%w: end must be after start, maximum 366 days", platform.ErrInvalid)
	}
	return a, b, nil
}

func Validate(items []Item) error {
	if len(items) == 0 || len(items) > 50 {
		return platform.ErrInvalid
	}
	for _, i := range items {
		if i.ModelID == "" || i.Quantity < 1 || i.Quantity > 100 {
			return platform.ErrInvalid
		}
		if _, _, err := Period(i.Start, i.End); err != nil {
			return err
		}
	}
	return nil
}

func Clone(r Reservation) Reservation {
	r.Items = append([]Quote(nil), r.Items...)
	for i := range r.Items {
		r.Items[i].UnitIDs = append([]string(nil), r.Items[i].UnitIDs...)
		r.Items[i].ReturnedUnitIDs = append([]string(nil), r.Items[i].ReturnedUnitIDs...)
	}
	return r
}
