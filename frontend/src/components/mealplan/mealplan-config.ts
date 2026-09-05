import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

// internal/config/defaults.go のフロント側 mirror。S09 で ConfigService が
// Wails 経由で Defaults() を返すまで、Go の Config struct と手動で同期する。
// source of truth の表は docs/product.md §1。
export const MEALPLAN_DEFAULTS = {
  defaultVisibleSlots: [Slot.SlotDinner] as string[],
  mealPlanCellVisibleItems: 2,
};

// カレンダーで上から下へ並べる slot 順。
export const SLOT_RENDER_ORDER = [Slot.SlotBreakfast, Slot.SlotLunch, Slot.SlotDinner] as const;
