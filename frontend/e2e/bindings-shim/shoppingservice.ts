import type * as shopping$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models.js";
import { call } from "./call.js";

export function List(periodStart: string, periodDays: number) {
  return call<shopping$0.List>("ShoppingService", "List", periodStart, periodDays);
}
