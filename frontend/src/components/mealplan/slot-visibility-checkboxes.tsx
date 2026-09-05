import { useTranslation } from "react-i18next";
import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { SLOT_RENDER_ORDER } from "./mealplan-config";

type Props = {
  value: Set<string>;
  onChange: (next: Set<string>) => void;
};

const LABEL_KEY: Record<string, string> = {
  [Slot.SlotBreakfast]: "pages.meal_plan.slot.breakfast",
  [Slot.SlotLunch]: "pages.meal_plan.slot.lunch",
  [Slot.SlotDinner]: "pages.meal_plan.slot.dinner",
};

export function SlotVisibilityCheckboxes({ value, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <fieldset className="inline-flex items-center gap-3 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm">
      <legend className="sr-only">{t("pages.meal_plan.slot_visibility.aria_label")}</legend>
      {SLOT_RENDER_ORDER.map((slot) => {
        const checked = value.has(slot);
        // 常に 1 個以上の slot を残すため、唯一 ON の checkbox は disable にする。
        const isLastOne = checked && value.size === 1;
        return (
          <label
            key={slot}
            className={`flex items-center gap-1.5 ${
              isLastOne ? "cursor-not-allowed text-neutral-400" : "cursor-pointer text-neutral-800"
            }`}
          >
            <input
              type="checkbox"
              className="h-4 w-4 accent-blue-600"
              checked={checked}
              disabled={isLastOne}
              onChange={(e) => {
                const next = new Set(value);
                if (e.target.checked) {
                  next.add(slot);
                } else {
                  next.delete(slot);
                }
                onChange(next);
              }}
            />
            {t(LABEL_KEY[slot])}
          </label>
        );
      })}
    </fieldset>
  );
}
