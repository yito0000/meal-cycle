package shopping

// ScaledAmount は (batch, recipe) 1 組の材料の量を世帯人数で換算し、換算したかを返す。
// 換算するのは同じ組の行が献立全体で 1 行だけの「1 回きりの献立」に限る。作り置きはレシピの量で何日分かを作る前提なので、
// 人数で掛けると日数分の量を見誤る (add-shopping-household-scaling design.md D1)。丸めは合算のあとで行う (D2)。
func ScaledAmount(amount float64, servings, household, pairRowCount int) (float64, bool) {
	if household <= 0 || amount == 0 || servings <= 0 || pairRowCount != 1 {
		return amount, false
	}
	return amount * float64(household) / float64(servings), true
}
