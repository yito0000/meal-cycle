import type * as mealplan$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models.js";
import { call } from "./call.js";

export function ListWeek(weekStart: string) {
  return call<mealplan$0.Entry[]>("MealPlanService", "ListWeek", weekStart);
}

export function AssignRecipe(date: string, slot: string, recipeID: string) {
  return call<mealplan$0.Entry>("MealPlanService", "AssignRecipe", date, slot, recipeID);
}

export function ReplaceEntry(date: string, slot: string, position: number, recipeID: string) {
  return call<void>("MealPlanService", "ReplaceEntry", date, slot, position, recipeID);
}

export function RemoveEntry(date: string, slot: string, position: number) {
  return call<void>("MealPlanService", "RemoveEntry", date, slot, position);
}

export function ClearSlot(date: string, slot: string) {
  return call<void>("MealPlanService", "ClearSlot", date, slot);
}

export function ReorderSlot(date: string, slot: string, orderedRecipeIDs: string[] | null) {
  return call<void>("MealPlanService", "ReorderSlot", date, slot, orderedRecipeIDs);
}

export function CopyLeftover(
  date: string,
  slot: string,
  days: number,
  position: number,
  mode: string,
  shareBatch: boolean,
) {
  return call<void>(
    "MealPlanService",
    "CopyLeftover",
    date,
    slot,
    days,
    position,
    mode,
    shareBatch,
  );
}

export function SplitBatch(date: string, slot: string, position: number) {
  return call<string>("MealPlanService", "SplitBatch", date, slot, position);
}

export function MergeBatch(date: string, slot: string, position: number, targetBatchID: string) {
  return call<void>("MealPlanService", "MergeBatch", date, slot, position, targetBatchID);
}

export function ListSlotBatches(date: string, slot: string, recipeID: string) {
  return call<mealplan$0.BatchSummary[]>(
    "MealPlanService",
    "ListSlotBatches",
    date,
    slot,
    recipeID,
  );
}
