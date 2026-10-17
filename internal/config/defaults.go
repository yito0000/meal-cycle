// Package config は meal-cycle のユーザー可変な既定値を集約する。
//
// docs/product.md §1 に可変にする値が列挙されている。Defaults() は既定値の source of truth で、
// S09 で利用者が変えた値は SettingsService がこの上に重ねて他サービスに渡す (add-settings-screen design.md D2/D3)。
//
// 呼び出し側の規約：「10 日」「3 日」のようなリテラル閾値を call site に書かない。
// 必ず Config 経由で読む。
package config

// Config は meal-cycle の実行時既定値。
// 将来 ConfigService がフロントに丸ごと返せるよう、flat かつ JSON serializable に保つ。
type Config struct {
	// UI 既定値 --------------------------------------------------------

	// 表示言語 ("ja" / "en")。Go は使わないが、保存と「既定に戻す」の経路を他の値と 1 本にするため持つ
	// (add-settings-screen design.md D7)。
	Language string `json:"language"`

	// S01 初回表示で見せる meal。internal/domain/mealplan.Meal と同じ文字列値を
	// []string で持つ (config が domain を import しないため)。
	DefaultVisibleMeals []string `json:"defaultVisibleMeals"`

	// 週プリセットが埋めるセル範囲。MVP はフリーテキスト、後で型付けする。
	WeeklyFillPattern string `json:"weeklyFillPattern"`

	// meal ごとの品数目標 {min, max}。強制ではなく、提案生成と UI コピー向けヒント。
	ItemsPerMealHint [2]int `json:"itemsPerMealHint"`

	// S01 の (date, meal) セルで折り畳まずに見せる品数の上限 ("他 N 件" 表示の閾値)。
	//
	// Deprecated: S09 実装時に MealPlanCellSizePresets に完全移行する。それまでの間は
	// 1 週 × 1 時間帯 (夕食のみ) プリセットの VisibleItems と同じ値を保つ。
	MealPlanCellVisibleItems int `json:"mealPlanCellVisibleItems"`

	// S01 のセル描画を「期間 × 表示時間帯数」で切り替えるためのプリセット表。
	// 未定義組み合わせに到達したときは最も密度が高い (4 週 × 3 時間帯) プリセットに fallback する。
	MealPlanCellSizePresets []CellSizePreset `json:"mealPlanCellSizePresets"`

	// S02 で保存直後の行をハイライトし続ける時間 (ms)。S04 → S02 復帰時に使う
	// (design.md D6)。
	RecipeListHighlightMs int `json:"recipeListHighlightMs"`

	// レシピ -----------------------------------------------------------

	// 量を持たない言葉。インポートで量 0 として読み、言葉を単位に残す。S09 で語彙を足せるよう可変値にする
	// (add-recipe-bulk-import design.md D5)。
	UnquantifiedWords []string `json:"unquantifiedWords"`

	// インポートするファイルが人数を省略したときの人数。世帯の人数に合わせたい値 (add-recipe-bulk-import design.md D9)。
	RecipeImportDefaultServings int `json:"recipeImportDefaultServings"`

	// 提案軸の閾値 -----------------------------------------------------

	// 「久しぶりに」軸から除外する直近利用日数。作り置き (同レシピを 2〜3 日
	// 続けて食べる) 運用を考慮した値。
	RecencyThresholdDays int `json:"recencyThresholdDays"`

	// 作り置きコピーで提示する日数プリセット。
	LeftoverCopyPresets []int `json:"leftoverCopyPresets"`

	// 週プリセットが跨ぐレシピ数の目標 {min, max}。
	WeeklyPresetRecipeCount [2]int `json:"weeklyPresetRecipeCount"`

	// 在庫アラート -----------------------------------------------------

	// 🔴 閾値 (N 日以内に賞味期限)。
	ExpiryAlertRedDays int `json:"expiryAlertRedDays"`

	// 🟡 閾値 (N 日以内、🔴 を超える範囲)。
	ExpiryAlertYellowDays int `json:"expiryAlertYellowDays"`

	// 買い物リスト -----------------------------------------------------

	// S07 で選べる対象期間の日数。S01 の 1 / 2 / 4 週 (7 / 14 / 28 日) から引き継げるよう、それらを必ず含める。
	ShoppingPeriodDayPresets []int `json:"shoppingPeriodDayPresets"`

	// S07 をサイドバーから開いたときの対象期間の日数 (今日から N 日間)。
	ShoppingDefaultPeriodDays int `json:"shoppingDefaultPeriodDays"`

	// 調理で量るための単位。買う量として意味を持たないので、S07 では量 0 の材料と同じく食材名だけで合算する
	// (add-measuring-units design.md D3)。表示では量の前に置く。
	MeasuringUnits []string `json:"measuringUnits"`
}

// CellSizePreset は S01 献立カレンダーの (date, meal) セル描画パラメータ。
// PeriodWeeks × VisibleMeals ごとに 1 行、セル高 / typography / 表示件数上限を持つ。
// FontSizeClass / MinHeightRem は Tailwind class 名に対応するが、JIT が拾えるように
// フロント側 (mealplan-config.ts) 冒頭で class 文字列をリテラル列挙してある。
type CellSizePreset struct {
	PeriodWeeks   int    `json:"periodWeeks"`
	VisibleMeals  int    `json:"visibleMeals"`
	MinHeightRem  int    `json:"minHeightRem"`
	FontSizeClass string `json:"fontSizeClass"`
	VisibleItems  int    `json:"visibleItems"`
}

// Defaults は主要ペルソナ (作り置き中心 / 夕食メイン) 向けにチューニング
// された既定値を返す (docs/product.md §1)。
func Defaults() Config {
	return Config{
		Language:                    "ja",
		DefaultVisibleMeals:         []string{"dinner"},
		WeeklyFillPattern:           "mon-fri-dinner",
		ItemsPerMealHint:            [2]int{1, 3},
		MealPlanCellVisibleItems:    4,
		RecipeListHighlightMs:       2000,
		RecencyThresholdDays:        10,
		LeftoverCopyPresets:         []int{2, 3, 4},
		WeeklyPresetRecipeCount:     [2]int{2, 3},
		ExpiryAlertRedDays:          3,
		ExpiryAlertYellowDays:       7,
		MealPlanCellSizePresets:     defaultCellSizePresets(),
		ShoppingPeriodDayPresets:    []int{3, 4, 7, 14, 28},
		ShoppingDefaultPeriodDays:   7,
		MeasuringUnits:              []string{"大さじ", "小さじ", "カップ"},
		UnquantifiedWords:           []string{"少々", "適量", "適宜", "少量", "ひとつまみ", "お好みで"},
		RecipeImportDefaultServings: 2,
	}
}

// defaultCellSizePresets は design.md D4 の 3×3 表を平坦配列に展開して返す。
// (期間 週数 × 表示時間帯数) → (min-h-*, text-*, 表示件数) の割り当ての source of truth。
func defaultCellSizePresets() []CellSizePreset {
	return []CellSizePreset{
		{PeriodWeeks: 1, VisibleMeals: 1, MinHeightRem: 48, FontSizeClass: "text-base", VisibleItems: 4},
		{PeriodWeeks: 1, VisibleMeals: 2, MinHeightRem: 40, FontSizeClass: "text-base", VisibleItems: 4},
		{PeriodWeeks: 1, VisibleMeals: 3, MinHeightRem: 32, FontSizeClass: "text-base", VisibleItems: 3},
		{PeriodWeeks: 2, VisibleMeals: 1, MinHeightRem: 32, FontSizeClass: "text-base", VisibleItems: 3},
		{PeriodWeeks: 2, VisibleMeals: 2, MinHeightRem: 24, FontSizeClass: "text-sm", VisibleItems: 3},
		{PeriodWeeks: 2, VisibleMeals: 3, MinHeightRem: 24, FontSizeClass: "text-sm", VisibleItems: 2},
		{PeriodWeeks: 4, VisibleMeals: 1, MinHeightRem: 20, FontSizeClass: "text-sm", VisibleItems: 2},
		{PeriodWeeks: 4, VisibleMeals: 2, MinHeightRem: 16, FontSizeClass: "text-sm", VisibleItems: 2},
		{PeriodWeeks: 4, VisibleMeals: 3, MinHeightRem: 16, FontSizeClass: "text-xs", VisibleItems: 2},
	}
}
