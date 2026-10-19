import { useTranslation } from "react-i18next";
import type { BatchSummary } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Meal } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { formatMealDate } from "../../lib/format-meal-date";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";

interface Props {
  open: boolean;
  candidates: BatchSummary[];
  onOpenChange: (open: boolean) => void;
  onSelect: (batchID: string) => void;
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

// 「バッチに合流する」を押したときの候補一覧。同一レシピ + ±dayRange 日の候補を表示する
// (spec の合流候補要件、design.md D7)。時間帯は絞らず、別時間帯の作り置きも候補に含めて
// item_label に 時間帯名を差し込む (proposal: add-mealplan-cross-meal-leftover)。
export function BatchMergeDialog({ open, candidates, onOpenChange, onSelect }: Props) {
  const { t, i18n } = useTranslation();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.batch.merge_dialog.title")}</DialogTitle>
        </DialogHeader>
        {candidates.length === 0 ? (
          <p className="text-sm text-neutral-600">
            {t("pages.meal_plan.batch.merge_dialog.empty")}
          </p>
        ) : (
          <ul className="flex max-h-64 flex-col gap-1 overflow-y-auto">
            {candidates.map((c) => (
              <li key={c.batchId}>
                <button
                  type="button"
                  className="block w-full cursor-pointer rounded-md border border-neutral-200 px-3 py-2 text-left text-sm hover:bg-neutral-50"
                  onClick={() => onSelect(c.batchId)}
                >
                  {t("pages.meal_plan.batch.merge_dialog.item_label", {
                    date: formatMealDate(c.date, i18n.language),
                    meal: t(mealLabelKey(c.meal)),
                    recipeName: c.recipeName,
                    count: c.memberCount,
                  })}
                </button>
              </li>
            ))}
          </ul>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.copy.conflict.cancel")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
