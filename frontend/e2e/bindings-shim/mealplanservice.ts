import type * as mealplan$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models.js";
import { call } from "./call.js";

export function ListPeriod(periodStart: string, periodWeeks: number) {
  return call<mealplan$0.Entry[]>("MealPlanService", "ListPeriod", periodStart, periodWeeks);
}

export function AssignRecipe(date: string, meal: string, recipeID: string) {
  return call<mealplan$0.Entry>("MealPlanService", "AssignRecipe", date, meal, recipeID);
}

export function ReplaceEntry(entryID: string, recipeID: string) {
  return call<void>("MealPlanService", "ReplaceEntry", entryID, recipeID);
}

export function RemoveEntry(entryID: string) {
  return call<mealplan$0.RemovedEntry>("MealPlanService", "RemoveEntry", entryID);
}

export function ClearMeal(date: string, meal: string) {
  return call<mealplan$0.RemovedEntry[]>("MealPlanService", "ClearMeal", date, meal);
}

export function RestoreEntries(entries: mealplan$0.RemovedEntry[] | null) {
  return call<void>("MealPlanService", "RestoreEntries", entries);
}

export function ReorderMeal(date: string, meal: string, orderedEntryIDs: string[] | null) {
  return call<void>("MealPlanService", "ReorderMeal", date, meal, orderedEntryIDs);
}

export function CopyLeftover(
  date: string,
  meal: string,
  days: number,
  mode: string,
  shareBatch: boolean,
) {
  return call<void>("MealPlanService", "CopyLeftover", date, meal, days, mode, shareBatch);
}

export function SplitBatch(entryID: string) {
  return call<string>("MealPlanService", "SplitBatch", entryID);
}

export function MergeBatch(entryID: string, targetBatchID: string) {
  return call<void>("MealPlanService", "MergeBatch", entryID, targetBatchID);
}

export function ListMergeCandidates(entryID: string) {
  return call<mealplan$0.BatchSummary[]>("MealPlanService", "ListMergeCandidates", entryID);
}

export function SpreadEntry(
  entryID: string,
  targets: mealplan$0.SpreadTarget[] | null,
  mode: string,
  shareBatch: boolean,
) {
  return call<void>("MealPlanService", "SpreadEntry", entryID, targets, mode, shareBatch);
}

export function MoveEntry(entryID: string, targetDate: string, targetMeal: string) {
  return call<void>("MealPlanService", "MoveEntry", entryID, targetDate, targetMeal);
}

export function SetCooked(entryID: string, cooked: boolean) {
  return call<void>("MealPlanService", "SetCooked", entryID, cooked);
}
