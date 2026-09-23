import { useTranslation } from "react-i18next";
import { DatePicker } from "@/components/ui/date-picker";
import { Label } from "@/components/ui/label";
import { SHOPPING_DEFAULTS } from "./shopping-config";
import type { ShoppingPeriod } from "./shopping-period";

type Props = {
  value: ShoppingPeriod;
  onChange: (next: ShoppingPeriod) => void;
};

export function ShoppingPeriodPicker({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-wrap items-center gap-4">
      <div className="flex items-center gap-2">
        <Label htmlFor="shopping-period-start">{t("pages.shopping.picker.start")}</Label>
        <DatePicker
          id="shopping-period-start"
          value={value.periodStart}
          onChange={(periodStart) => onChange({ ...value, periodStart })}
        />
      </div>
      <fieldset className="inline-flex items-center rounded-md border border-neutral-200 bg-white text-sm">
        <legend className="sr-only">{t("pages.shopping.picker.days")}</legend>
        {SHOPPING_DEFAULTS.periodDayPresets.map((days) => {
          const isActive = days === value.periodDays;
          return (
            <button
              key={days}
              type="button"
              onClick={() => onChange({ ...value, periodDays: days })}
              aria-pressed={isActive}
              data-testid={`shopping-days-${days}`}
              className={`px-3 py-1.5 first:rounded-l-md last:rounded-r-md transition-colors ${
                isActive ? "bg-blue-600 text-white" : "text-neutral-800 hover:bg-neutral-100"
              }`}
            >
              {t("pages.shopping.picker.day_option", { count: days })}
            </button>
          );
        })}
      </fieldset>
    </div>
  );
}
