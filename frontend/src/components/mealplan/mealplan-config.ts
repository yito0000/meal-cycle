import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

// Tailwind v4 JIT が拾えるよう、CellSizePreset で使う class を静的リテラルで事前列挙する。
// 動的補間 (`min-h-${n}`) は JIT の抽出対象から外れやすい (design.md D1 Risks 参照)。
// 対象: min-h-16 min-h-20 min-h-24 min-h-32 min-h-40 min-h-48 / text-xs text-sm text-base
// prettier-ignore
const _TAILWIND_JIT_KEEP =
  "min-h-16 min-h-20 min-h-24 min-h-32 min-h-40 min-h-48 text-xs text-sm text-base";
void _TAILWIND_JIT_KEEP;

// internal/config/defaults.go のフロント側 mirror。S09 で ConfigService が
// Wails 経由で Defaults() を返すまで、Go の Config struct と手動で同期する。
// source of truth の表は docs/product.md §1。
export const MEALPLAN_DEFAULTS = {
  defaultVisibleMeals: [Meal.MealDinner] as string[],
  // Deprecated: 参照は preset.visibleItems 経由に移行する (S09 で ConfigService に完全移行)。
  // 1 週 × 1 時間帯 プリセットの visibleItems と同じ値を保つ。
  mealPlanCellVisibleItems: 4,
  leftoverCopyPresets: [2, 3, 4] as const,
};

// カレンダーで上から下へ並べる時間帯順。
export const MEAL_RENDER_ORDER = [Meal.MealBreakfast, Meal.MealLunch, Meal.MealDinner] as const;

// 複数行バッチを見分ける枠色 (Tailwind class)。design.md D5 で採用: HSL ハッシュではなく
// 事前定義の 3 色ローテーションで、1 週間内 3〜4 バッチまでの識別を目的にする。
// 4 バッチ以上あると同じ色が回るが、位置が離れていれば混同されにくい。
export const BATCH_COLORS = [
  "border-orange-400 bg-orange-50",
  "border-emerald-400 bg-emerald-50",
  "border-violet-400 bg-violet-50",
] as const;

export type CellSizePreset = {
  periodWeeks: number;
  visibleMeals: number;
  minHeightRem: number;
  fontSizeClass: "text-xs" | "text-sm" | "text-base";
  visibleItems: number;
};

// internal/config/defaults.go の defaultCellSizePresets() の mirror。両方同時に更新する。
export const CELL_SIZE_PRESETS: readonly CellSizePreset[] = [
  {
    periodWeeks: 1,
    visibleMeals: 1,
    minHeightRem: 48,
    fontSizeClass: "text-base",
    visibleItems: 4,
  },
  {
    periodWeeks: 1,
    visibleMeals: 2,
    minHeightRem: 40,
    fontSizeClass: "text-base",
    visibleItems: 4,
  },
  {
    periodWeeks: 1,
    visibleMeals: 3,
    minHeightRem: 32,
    fontSizeClass: "text-base",
    visibleItems: 3,
  },
  {
    periodWeeks: 2,
    visibleMeals: 1,
    minHeightRem: 32,
    fontSizeClass: "text-base",
    visibleItems: 3,
  },
  { periodWeeks: 2, visibleMeals: 2, minHeightRem: 24, fontSizeClass: "text-sm", visibleItems: 3 },
  { periodWeeks: 2, visibleMeals: 3, minHeightRem: 24, fontSizeClass: "text-sm", visibleItems: 2 },
  { periodWeeks: 4, visibleMeals: 1, minHeightRem: 20, fontSizeClass: "text-sm", visibleItems: 2 },
  { periodWeeks: 4, visibleMeals: 2, minHeightRem: 16, fontSizeClass: "text-sm", visibleItems: 2 },
  { periodWeeks: 4, visibleMeals: 3, minHeightRem: 16, fontSizeClass: "text-xs", visibleItems: 2 },
] as const;

// 未定義組み合わせは最も密度が高い (4 週 × 3 時間帯) プリセットに落とす。log なし。
const FALLBACK_PRESET: CellSizePreset = CELL_SIZE_PRESETS[CELL_SIZE_PRESETS.length - 1];

export function pickCellPreset(periodWeeks: number, visibleMeals: number): CellSizePreset {
  return (
    CELL_SIZE_PRESETS.find(
      (p) => p.periodWeeks === periodWeeks && p.visibleMeals === visibleMeals,
    ) ?? FALLBACK_PRESET
  );
}

// 事前列挙した Tailwind class 名にプリセット値を割り当てるためのマップ。
// class を動的補間で組み立てず必ずリテラルで参照するため、ここで switch する。
export function minHeightClass(preset: CellSizePreset): string {
  switch (preset.minHeightRem) {
    case 16:
      return "min-h-16";
    case 20:
      return "min-h-20";
    case 24:
      return "min-h-24";
    case 32:
      return "min-h-32";
    case 40:
      return "min-h-40";
    case 48:
      return "min-h-48";
    default:
      return "min-h-16";
  }
}
