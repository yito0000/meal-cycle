import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

// Frontend mirror of internal/config/defaults.go for the meal-plan UI.
//
// These constants MUST stay in sync with the Go Config struct until a
// ConfigService exposes Defaults() over Wails (planned with S09). See
// docs/product.md §1 for the source-of-truth table.
export const MEALPLAN_DEFAULTS = {
  defaultVisibleSlots: [Slot.SlotDinner] as string[],
  mealPlanCellVisibleItems: 2,
};

// Order slots render top-to-bottom in the calendar (breakfast → dinner).
export const SLOT_RENDER_ORDER = [Slot.SlotBreakfast, Slot.SlotLunch, Slot.SlotDinner] as const;
