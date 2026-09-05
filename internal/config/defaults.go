// Package config は meal-cycle のユーザー可変な既定値を集約する。
//
// docs/product.md §1 に「本来 S09 設定画面で可変にする値」が列挙されており、
// MVP 期は Defaults() が固定定数として返す。設定 UI が入っても呼び出し側は
// Config を読むままにしておき、内部だけを settings store 読み込みに差し替える。
//
// 呼び出し側の規約：「10 日」「3 日」のようなリテラル閾値を call site に書かない。
// 必ず Config 経由で読む。
package config

// Config は meal-cycle の実行時既定値。
// 将来 ConfigService がフロントに丸ごと返せるよう、flat かつ JSON serializable に保つ。
type Config struct {
	// UI 既定値 --------------------------------------------------------

	// S01 初回表示で見せる slot。internal/domain/mealplan.Slot と同じ文字列値を
	// []string で持つ (config が domain を import しないため)。
	DefaultVisibleSlots []string `json:"defaultVisibleSlots"`

	// 週プリセットが埋めるセル範囲。MVP はフリーテキスト、後で型付けする。
	WeeklyFillPattern string `json:"weeklyFillPattern"`

	// slot ごとの品数目標 {min, max}。強制ではなく、提案生成と UI コピー向けヒント。
	ItemsPerSlotHint [2]int `json:"itemsPerSlotHint"`

	// S01 の (date, slot) セルで折り畳まずに見せる品数の上限 ("他 N 件" 表示の閾値)。
	MealPlanCellVisibleItems int `json:"mealPlanCellVisibleItems"`

	// S02 で保存直後の行をハイライトし続ける時間 (ms)。S04 → S02 復帰時に使う
	// (design.md D6)。
	RecipeListHighlightMs int `json:"recipeListHighlightMs"`

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
}

// Defaults は主要ペルソナ (作り置き中心 / 夕食メイン) 向けにチューニング
// された既定値を返す (docs/product.md §1)。
func Defaults() Config {
	return Config{
		DefaultVisibleSlots:      []string{"dinner"},
		WeeklyFillPattern:        "mon-fri-dinner",
		ItemsPerSlotHint:         [2]int{1, 3},
		MealPlanCellVisibleItems: 2,
		RecipeListHighlightMs:    2000,
		RecencyThresholdDays:     10,
		LeftoverCopyPresets:      []int{2, 3, 4},
		WeeklyPresetRecipeCount:  [2]int{2, 3},
		ExpiryAlertRedDays:       3,
		ExpiryAlertYellowDays:    7,
	}
}
