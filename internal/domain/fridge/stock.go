// Package fridge は fridge capability (S05 / S06) のドメイン層。
package fridge

// ExpiryStatus は賞味期限までの残り日数から決まる状態。S05 の色ドット / 期限切れバッジに対応する。
type ExpiryStatus string

const (
	ExpiryNone   ExpiryStatus = "none"
	ExpiryFresh  ExpiryStatus = "fresh"
	ExpirySoon   ExpiryStatus = "soon"
	ExpiryUrgent ExpiryStatus = "urgent"
	// 当日 (残り 0 日) は今日中なら使えるので Urgent に残し、残り日数が負のときだけ Expired にする (design.md D14)。
	ExpiryExpired ExpiryStatus = "expired"
)

// StockItem は冷蔵庫在庫 1 行。
//
// ExpiresAt は YYYY-MM-DD、空文字が「期限なし」(DB 上は NULL)。フォームの
// <input type="date"> と同じ表現にして変換を DB 境界 1 箇所に閉じる (design.md D3)。
// AddedAt / UpdatedAt は RFC3339 (UTC)。ExpiryStatus / DaysLeft は読み出し時の
// 算出値で、入力側の値は無視される。
type StockItem struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Amount       float64      `json:"amount"`
	Unit         string       `json:"unit"`
	ExpiresAt    string       `json:"expiresAt"`
	AddedAt      string       `json:"addedAt"`
	UpdatedAt    string       `json:"updatedAt"`
	ExpiryStatus ExpiryStatus `json:"expiryStatus"`
	DaysLeft     *int         `json:"daysLeft"`
}

// NameSuggestion は食材名の候補 1 件と、その名前で直近に使われた単位。
type NameSuggestion struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
}

// Suggestions は S06 の食材名 / 単位入力の候補。
type Suggestions struct {
	Names []NameSuggestion `json:"names"`
	Units []string         `json:"units"`
}
