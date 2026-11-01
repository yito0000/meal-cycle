import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type {
  PlanItem,
  SpreadResult,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import type {
  WeekPlan,
  WeekPlanItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/suggestion/models";
import {
  MealPlanService,
  SuggestionService,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { formatDishDates } from "../shopping/format-dish-dates";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { errorI18nKey } from "./error-messages";

type Props = {
  open: boolean;
  weekStart: string;
  onOpenChange: (open: boolean) => void;
  onApplied: (result: SpreadResult, count: number) => void;
};

// 案の組ごとに Group を振り、同じ組の行を 1 つの作り置きにする (add-mealplan-suggest-week-preset design.md D5)。
function toPlanItems(plan: WeekPlan): PlanItem[] {
  return (plan.items ?? []).flatMap((it, group) =>
    (it.dates ?? []).map((date) => ({
      date,
      meal: it.meal as PlanItem["meal"],
      recipeId: it.recipeId,
      group,
    })),
  );
}

export function WeekPlanDialog({ open, weekStart, onOpenChange, onApplied }: Props) {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);
  const { data, isLoading, isError } = useQuery({
    queryKey: ["suggestions", "week", weekStart],
    queryFn: () => SuggestionService.WeekPlans(weekStart),
    enabled: open,
    staleTime: 0,
  });
  const slots = data?.slots ?? [];
  const plans = data?.plans ?? [];

  const apply = async (plan: WeekPlan) => {
    if (pending) return;
    const items = toPlanItems(plan);
    setPending(true);
    try {
      const result = await MealPlanService.ApplyPlan(items);
      onApplied(result, items.length);
    } catch (err) {
      toast.error(t("pages.meal_plan.week_plan.apply_failed"), {
        description: t(errorI18nKey(err)),
      });
    } finally {
      setPending(false);
    }
  };

  const reason = (it: WeekPlanItem): string => {
    const parts: string[] = [];
    const stocks = it.stocks ?? [];
    if (stocks.length > 0) {
      const names = stocks
        .map((s) => s.name)
        .join(t("pages.meal_plan.picker.suggestion.name_separator"));
      const days = stocks[0].daysLeft;
      parts.push(
        days === 0
          ? t("pages.meal_plan.picker.suggestion.by_stock_reason_today", { names })
          : t("pages.meal_plan.picker.suggestion.by_stock_reason", { names, days }),
      );
    }
    if (it.daysSince > 0) {
      parts.push(t("pages.meal_plan.picker.suggestion.by_recency_reason", { days: it.daysSince }));
    }
    return parts.join(" / ");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] max-w-2xl flex-col">
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.week_plan.title")}</DialogTitle>
          <DialogDescription>{t("pages.meal_plan.week_plan.description")}</DialogDescription>
        </DialogHeader>
        <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
          {isLoading && <p className="text-sm text-neutral-500">{t("pages.meal_plan.loading")}</p>}
          {isError && (
            <p className="text-sm text-red-700">{t("pages.meal_plan.week_plan.load_error")}</p>
          )}
          {data && slots.length === 0 && (
            <p className="text-sm text-neutral-500" data-testid="week-plan-no-slots">
              {t("pages.meal_plan.week_plan.no_slots")}
            </p>
          )}
          {data && slots.length > 0 && plans.length === 0 && (
            <p className="text-sm text-neutral-500">{t("pages.meal_plan.week_plan.no_plans")}</p>
          )}
          <div className="flex flex-col gap-3">
            {plans.map((plan) => (
              <section
                key={plan.kind}
                data-testid="week-plan"
                data-kind={plan.kind}
                aria-label={t(`pages.meal_plan.week_plan.kind.${plan.kind}`)}
                className="flex flex-col gap-2 rounded-md border border-neutral-200 p-3"
              >
                <div className="flex items-center justify-between gap-2">
                  <h3 className="text-sm font-semibold">
                    {t(`pages.meal_plan.week_plan.kind.${plan.kind}`)}
                  </h3>
                  <Button type="button" size="sm" disabled={pending} onClick={() => apply(plan)}>
                    {t("pages.meal_plan.week_plan.apply")}
                  </Button>
                </div>
                <ul className="flex flex-col gap-1">
                  {(plan.items ?? []).map((it) => (
                    <li
                      key={it.recipeId}
                      data-testid="week-plan-item"
                      className="grid grid-cols-[6rem_1fr] gap-x-3 text-sm"
                    >
                      <span className="tabular-nums text-neutral-600">
                        {formatDishDates(it.dates ?? [], t)}
                      </span>
                      <span>
                        <span className="block">{it.recipeName}</span>
                        {reason(it) && (
                          <span className="block text-xs text-neutral-500">{reason(it)}</span>
                        )}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.week_plan.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
