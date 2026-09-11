import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

// internal/config/defaults.go のフロント側 mirror。S09 で ConfigService が
// Wails 経由で Defaults() を返すまで、Go の Config struct と手動で同期する。
// source of truth の表は docs/product.md §1。
export const MEALPLAN_DEFAULTS = {
  defaultVisibleSlots: [Slot.SlotDinner] as string[],
  mealPlanCellVisibleItems: 2,
  leftoverCopyPresets: [2, 3, 4] as const,
};

// カレンダーで上から下へ並べる slot 順。
export const SLOT_RENDER_ORDER = [Slot.SlotBreakfast, Slot.SlotLunch, Slot.SlotDinner] as const;

// 複数行バッチを見分ける枠色 (Tailwind class)。design.md D5 で採用: HSL ハッシュではなく
// 事前定義の 3 色ローテーションで、1 週間内 3〜4 バッチまでの識別を目的にする。
// 4 バッチ以上あると同じ色が回るが、位置が離れていれば混同されにくい。
export const BATCH_COLORS = [
  "border-orange-400 bg-orange-50",
  "border-emerald-400 bg-emerald-50",
  "border-violet-400 bg-violet-50",
] as const;
