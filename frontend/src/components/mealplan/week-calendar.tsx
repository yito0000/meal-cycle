import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Slot } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanCell } from "./meal-plan-cell";
import { SLOT_RENDER_ORDER } from "./mealplan-config";
import { formatDate, isSameLocalDate, weekDates } from "./week-utils";

type Props = {
  weekStart: Date;
  entries: Entry[];
  visibleSlots: Set<string>;
};

function slotLabelKey(slot: string): string {
  switch (slot) {
    case Slot.SlotBreakfast:
      return "pages.meal_plan.slot.breakfast";
    case Slot.SlotLunch:
      return "pages.meal_plan.slot.lunch";
    default:
      return "pages.meal_plan.slot.dinner";
  }
}

export function WeekCalendar({ weekStart, entries, visibleSlots }: Props) {
  const { t, i18n } = useTranslation();
  const dates = weekDates(weekStart);
  const today = new Date();
  const slots = SLOT_RENDER_ORDER.filter((s) => visibleSlots.has(s));

  const grouped = new Map<string, Entry[]>();
  for (const e of entries) {
    const key = `${e.date}::${e.slot}`;
    const bucket = grouped.get(key);
    if (bucket) {
      bucket.push(e);
    } else {
      grouped.set(key, [e]);
    }
  }

  const weekdayFmt = new Intl.DateTimeFormat(i18n.language, { weekday: "short" });
  const dateFmt = new Intl.DateTimeFormat(i18n.language, { month: "numeric", day: "numeric" });

  return (
    <div
      className="grid gap-1 text-sm"
      style={{ gridTemplateColumns: "6rem repeat(7, minmax(0, 1fr))" }}
    >
      <div />
      {dates.map((d) => {
        const isToday = isSameLocalDate(d, today);
        return (
          <div
            key={d.toISOString()}
            className={`flex flex-col items-center rounded-md p-2 text-center ${
              isToday ? "bg-blue-50 text-blue-900" : "text-neutral-700"
            }`}
            aria-current={isToday ? "date" : undefined}
          >
            <span className="text-xs text-neutral-500">{weekdayFmt.format(d)}</span>
            <span className="text-base font-semibold">{dateFmt.format(d)}</span>
            {isToday && (
              <span className="text-[10px] uppercase tracking-wide text-blue-700">
                {t("pages.meal_plan.today")}
              </span>
            )}
          </div>
        );
      })}

      {slots.map((slot) => (
        <div key={`row-${slot}`} className="contents">
          <div className="flex items-center justify-end pr-2 text-xs font-medium text-neutral-500">
            {t(slotLabelKey(slot))}
          </div>
          {dates.map((d) => {
            const key = `${formatDate(d)}::${slot}`;
            return <MealPlanCell key={key} entries={grouped.get(key) ?? []} />;
          })}
        </div>
      ))}
    </div>
  );
}
