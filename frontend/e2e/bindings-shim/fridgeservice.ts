import type * as fridge$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models.js";
import { call } from "./call.js";

export function Add(input: fridge$0.StockItem) {
  return call<fridge$0.StockItem>("FridgeService", "Add", input);
}

export function Delete(id: string) {
  return call<void>("FridgeService", "Delete", id);
}

export function List() {
  return call<fridge$0.StockItem[]>("FridgeService", "List");
}

export function ListSuggestions() {
  return call<fridge$0.Suggestions>("FridgeService", "ListSuggestions");
}

export function Restore(item: fridge$0.StockItem) {
  return call<fridge$0.StockItem>("FridgeService", "Restore", item);
}

export function Update(id: string, input: fridge$0.StockItem) {
  return call<fridge$0.StockItem>("FridgeService", "Update", id, input);
}
