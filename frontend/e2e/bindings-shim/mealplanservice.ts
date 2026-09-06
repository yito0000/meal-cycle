import type * as mealplan$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models.js";
import { call } from "./call.js";

export function ListWeek(weekStart: string) {
  return call<mealplan$0.Entry[] | null>("MealPlanService", "ListWeek", weekStart);
}
