package mealplan

// SpreadGrid は「作り置きを広げる」の格子を描くのに要る値。検証の上限と既存の品を判定する範囲を
// 同じ値から決めるため、1 回の呼び出しで返す (revise-settings-constraints design.md D2)。
type SpreadGrid struct {
	// 起点日から何日後まで選べるか。
	MaxOffsetDays int `json:"maxOffsetDays"`
	// 起点日から MaxOffsetDays+1 日分の献立。
	Entries []Entry `json:"entries"`
}
