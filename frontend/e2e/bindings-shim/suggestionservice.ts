import type * as suggestion$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/suggestion/models.js";
import { call } from "./call.js";

export function List() {
  return call<suggestion$0.Result>("SuggestionService", "List");
}

export function WeekPlans(weekStart: string) {
  return call<suggestion$0.WeekPlanResult>("SuggestionService", "WeekPlans", weekStart);
}
