package entity

const (
	OrderPending        = "pending"
	OrderConfirmed      = "confirmed"
	OrderActive         = "active"
	OrderCompleted      = "completed"
	OrderCancelled      = "cancelled"
	OrderRejected       = "rejected"
	OrderCancelling     = "cancelling"
	OrderIssuing        = "issuing"
	OrderReturning      = "returning"
	ActionReserve       = "reserve"
	ActionCancel        = "cancel"
	ActionIssue         = "issue"
	ActionReturn        = "return"
	ReservationReserved = "reserved"
)

type Quote struct {
	Item            Item
	Name            string
	DailyPrice      int64
	Total           int64
	UnitIDs         []string
	ReturnedUnitIDs []string
}
type Reservation struct {
	OrderID string
	State   string
	Items   []Quote
}
type Event struct {
	Action     string
	ActorID    string
	Reason     string
	OccurredAt string
}
type Order struct {
	Version       int64
	ID            string
	UserID        string
	State         string
	Error         string
	RequestID     string
	Items         []Quote
	Total         int64
	History       []Event
	PendingAction string
	PendingActor  string
	PendingReason string
	PendingUnits  []string
	CartVersion   uint64
}

func Clone(o Order) Order {
	o.Items = append([]Quote(nil), o.Items...)
	for i := range o.Items {
		o.Items[i].UnitIDs = append([]string(nil), o.Items[i].UnitIDs...)
		o.Items[i].ReturnedUnitIDs = append([]string(nil), o.Items[i].ReturnedUnitIDs...)
	}
	o.History = append([]Event(nil), o.History...)
	o.PendingUnits = append([]string(nil), o.PendingUnits...)
	return o
}
