package settings

import "slices"

// Constraints は S09 の検証に結びつく規則のうち、フロントが表示に使う値。
// フロントに写しを置くと Go と片方だけ直る事故が起きるので、Go から返す (revise-settings-constraints design.md D1)。
type Constraints struct {
	ShoppingRequiredDays []int `json:"shoppingRequiredDays"`
	// 既定の日数を候補から消したときに選び直す日数。消せない日数に含まれていれば必ず候補に残る。
	ShoppingFallbackDays int `json:"shoppingFallbackDays"`
}

func CurrentConstraints() Constraints {
	return Constraints{
		ShoppingRequiredDays: slices.Clone(ShoppingRequiredDays),
		ShoppingFallbackDays: ShoppingRequiredDays[0],
	}
}
