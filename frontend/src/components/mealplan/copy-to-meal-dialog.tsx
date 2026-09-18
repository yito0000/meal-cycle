import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { Label } from "../ui/label";
import { MEAL_RENDER_ORDER } from "./mealplan-config";

interface Props {
  open: boolean;
  sourceDate: string;
  sourceMeal: string;
  dayRange: number;
  onOpenChange: (open: boolean) => void;
  onSelect: (targetDate: string, targetMeal: string) => void;
}

function mealLabelKey(meal: string): string {
  switch (meal) {
    case Meal.MealBreakfast:
      return "pages.meal_plan.meal.breakfast";
    case Meal.MealLunch:
      return "pages.meal_plan.meal.lunch";
    default:
      return "pages.meal_plan.meal.dinner";
  }
}

function addDaysISO(date: string, days: number): string {
  const [y, m, d] = date.split("-").map(Number);
  const t = new Date(y, m - 1, d + days);
  const yy = t.getFullYear();
  const mm = String(t.getMonth() + 1).padStart(2, "0");
  const dd = String(t.getDate()).padStart(2, "0");
  return `${yy}-${mm}-${dd}`;
}

// 「別の時間帯にも作り置きを追加」ダイアログ。対象日は起点日から ±dayRange 日以内、
// 対象時間帯は元と異なる時間帯のみを選択可にする (defense in depth の一段、サーバ側でも再度検証)。
// 対象日の既定は「起点日 + 1 日」、対象時間帯の既定は「昼食」(現実の使い方に一致)。
export function CopyToMealDialog({
  open,
  sourceDate,
  sourceMeal,
  dayRange,
  onOpenChange,
  onSelect,
}: Props) {
  const { t } = useTranslation();
  const availableMeals = useMemo(
    () => MEAL_RENDER_ORDER.filter((m) => m !== sourceMeal),
    [sourceMeal],
  );
  const defaultTargetDate = useMemo(() => addDaysISO(sourceDate, 1), [sourceDate]);
  const defaultTargetMeal = useMemo(() => {
    if (availableMeals.includes(Meal.MealLunch)) return Meal.MealLunch as string;
    return availableMeals[0] ?? "";
  }, [availableMeals]);

  const [targetDate, setTargetDate] = useState(defaultTargetDate);
  const [targetMeal, setTargetMeal] = useState<string>(defaultTargetMeal);

  useEffect(() => {
    if (open) {
      setTargetDate(defaultTargetDate);
      setTargetMeal(defaultTargetMeal);
    }
  }, [open, defaultTargetDate, defaultTargetMeal]);

  const minDate = useMemo(() => addDaysISO(sourceDate, -dayRange), [sourceDate, dayRange]);
  const maxDate = useMemo(() => addDaysISO(sourceDate, dayRange), [sourceDate, dayRange]);
  const canSubmit = availableMeals.length > 0 && targetMeal !== sourceMeal && targetMeal !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.copy_to_meal.title")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3 text-sm">
          <div className="flex flex-col gap-1">
            <Label htmlFor="copy-to-meal-date">
              {t("pages.meal_plan.copy_to_meal.target_date_label")}
            </Label>
            <input
              id="copy-to-meal-date"
              type="date"
              value={targetDate}
              min={minDate}
              max={maxDate}
              onChange={(e) => setTargetDate(e.target.value)}
              className="rounded-md border border-neutral-300 bg-white px-3 py-1 text-sm shadow-sm"
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label>{t("pages.meal_plan.copy_to_meal.target_meal_label")}</Label>
            {availableMeals.length === 0 ? (
              <p className="text-xs text-neutral-500">
                {t("pages.meal_plan.copy_to_meal.empty_meals")}
              </p>
            ) : (
              <div className="flex gap-2">
                {availableMeals.map((m) => (
                  <label
                    key={m}
                    className={`cursor-pointer rounded-md border px-3 py-1 text-sm ${
                      targetMeal === m
                        ? "border-blue-500 bg-blue-50 text-blue-800"
                        : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
                    }`}
                  >
                    <input
                      type="radio"
                      name="copy-to-meal-target"
                      className="sr-only"
                      value={m}
                      checked={targetMeal === m}
                      onChange={() => setTargetMeal(m)}
                    />
                    {t(mealLabelKey(m))}
                  </label>
                ))}
              </div>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.copy_to_meal.cancel")}
          </Button>
          <Button
            disabled={!canSubmit}
            onClick={() => {
              if (!canSubmit) return;
              onSelect(targetDate, targetMeal);
            }}
          >
            {t("pages.meal_plan.copy_to_meal.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
