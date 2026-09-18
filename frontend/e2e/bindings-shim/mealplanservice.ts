import type * as mealplan$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models.js";
import { call } from "./call.js";

export function ListPeriod(periodStart: string, periodWeeks: number) {
  return call<mealplan$0.Entry[]>("MealPlanService", "ListPeriod", periodStart, periodWeeks);
}

export function AssignRecipe(date: string, meal: string, recipeID: string) {
  return call<mealplan$0.Entry>("MealPlanService", "AssignRecipe", date, meal, recipeID);
}

export function ReplaceEntry(date: string, meal: string, position: number, recipeID: string) {
  return call<void>("MealPlanService", "ReplaceEntry", date, meal, position, recipeID);
}

export function RemoveEntry(date: string, meal: string, position: number) {
  return call<void>("MealPlanService", "RemoveEntry", date, meal, position);
}

export function ClearMeal(date: string, meal: string) {
  return call<void>("MealPlanService", "ClearMeal", date, meal);
}

export function ReorderMeal(date: string, meal: string, orderedRecipeIDs: string[] | null) {
  return call<void>("MealPlanService", "ReorderMeal", date, meal, orderedRecipeIDs);
}

export function CopyLeftover(
  date: string,
  meal: string,
  days: number,
  position: number,
  mode: string,
  shareBatch: boolean,
) {
  return call<void>(
    "MealPlanService",
    "CopyLeftover",
    date,
    meal,
    days,
    position,
    mode,
    shareBatch,
  );
}

export function SplitBatch(date: string, meal: string, position: number) {
  return call<string>("MealPlanService", "SplitBatch", date, meal, position);
}

export function MergeBatch(date: string, meal: string, position: number, targetBatchID: string) {
  return call<void>("MealPlanService", "MergeBatch", date, meal, position, targetBatchID);
}

export function ListMergeCandidates(date: string, meal: string, recipeID: string) {
  return call<mealplan$0.BatchSummary[]>(
    "MealPlanService",
    "ListMergeCandidates",
    date,
    meal,
    recipeID,
  );
}

export function CopyToOtherMeal(
  sourceDate: string,
  sourceMeal: string,
  sourcePosition: number,
  targetDate: string,
  targetMeal: string,
  mode: string,
) {
  return call<void>(
    "MealPlanService",
    "CopyToOtherMeal",
    sourceDate,
    sourceMeal,
    sourcePosition,
    targetDate,
    targetMeal,
    mode,
  );
}
