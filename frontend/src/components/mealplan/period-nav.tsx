import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { formatDate, weekStartOf } from "./week-utils";

export type PeriodWeeks = 1 | 2 | 4;

type Props = {
  periodStart: Date;
  periodWeeks: PeriodWeeks;
  onPeriodChange: (start: Date, weeks: PeriodWeeks) => void;
  onToday: () => void;
};

const PERIOD_OPTIONS: readonly PeriodWeeks[] = [1, 2, 4];

// 期間長ぶんの日数シフト。design.md D3: `＜` / `＞` は現在の期間長単位でシフトする。
function shiftDays(start: Date, days: number): Date {
  return new Date(start.getFullYear(), start.getMonth(), start.getDate() + days);
}

export function PeriodNav({ periodStart, periodWeeks, onPeriodChange, onToday }: Props) {
  const { t } = useTranslation();
  const shift = periodWeeks * 7;

  const goPrev = () => onPeriodChange(shiftDays(periodStart, -shift), periodWeeks);
  const goNext = () => onPeriodChange(shiftDays(periodStart, shift), periodWeeks);
  const goToday = () => {
    // 「今週」= 今日を含む週の月曜。onToday 側で fresh Date を作らないと同じ instance を返してしまう。
    const today = weekStartOf(new Date());
    onPeriodChange(today, periodWeeks);
    onToday();
  };

  return (
    <div className="flex items-center gap-2">
      <fieldset className="inline-flex items-center gap-1 rounded-md border border-neutral-200 bg-white px-1 py-1 text-sm">
        <legend className="sr-only">{t("pages.meal_plan.period.aria_label")}</legend>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={goPrev}
          aria-label={t("pages.meal_plan.period.prev")}
          data-testid="period-prev"
        >
          ＜
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={goToday}
          data-testid="period-today"
          data-period-start={formatDate(periodStart)}
        >
          {t("pages.meal_plan.period.today")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={goNext}
          aria-label={t("pages.meal_plan.period.next")}
          data-testid="period-next"
        >
          ＞
        </Button>
      </fieldset>

      <fieldset className="inline-flex items-center rounded-md border border-neutral-200 bg-white text-sm">
        <legend className="sr-only">{t("pages.meal_plan.period.length_aria_label")}</legend>
        {PERIOD_OPTIONS.map((w) => {
          const isActive = w === periodWeeks;
          return (
            <button
              key={w}
              type="button"
              onClick={() => onPeriodChange(periodStart, w)}
              aria-pressed={isActive}
              data-testid={`period-weeks-${w}`}
              className={`px-3 py-1.5 first:rounded-l-md last:rounded-r-md transition-colors ${
                isActive ? "bg-blue-600 text-white" : "text-neutral-800 hover:bg-neutral-100"
              }`}
            >
              {t(`pages.meal_plan.period.week_${w}`)}
            </button>
          );
        })}
      </fieldset>
    </div>
  );
}
