package fridge

// StapleState は常備品の状態。値の大小に意味は無く、並び順は利用側で決める (add-fridge-staples design.md D2)。
type StapleState string

const (
	StapleEnough StapleState = "enough"
	StapleLow    StapleState = "low"
	StapleOut    StapleState = "out"
)

func (s StapleState) Valid() bool {
	switch s {
	case StapleEnough, StapleLow, StapleOut:
		return true
	}
	return false
}

// Staple は量を持たず状態だけを持つ在庫 1 品。AddedAt / UpdatedAt は RFC3339 (UTC)。
type Staple struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	State     StapleState `json:"state"`
	AddedAt   string      `json:"addedAt"`
	UpdatedAt string      `json:"updatedAt"`
}
