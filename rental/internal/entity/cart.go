package entity

type Item struct {
	ModelID  string
	Quantity int32
	Start    string
	End      string
}
type Cart struct {
	Items   []Item
	Version uint64
}
