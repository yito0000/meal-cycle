import type * as fridge$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models.js";
import { call } from "./call.js";

export function Add(input: fridge$0.StockItem) {
  return call<fridge$0.StockItem>("FridgeService", "Add", input);
}

export function AddMany(inputs: fridge$0.StockItem[] | null) {
  return call<fridge$0.StockItem[]>("FridgeService", "AddMany", inputs);
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

export function ListStaples() {
  return call<fridge$0.Staple[]>("FridgeService", "ListStaples");
}

export function AddStaple(name: string) {
  return call<fridge$0.Staple>("FridgeService", "AddStaple", name);
}

export function SetStapleState(id: string, state: fridge$0.StapleState) {
  return call<fridge$0.Staple>("FridgeService", "SetStapleState", id, state);
}

export function DeleteStaple(id: string) {
  return call<void>("FridgeService", "DeleteStaple", id);
}

export function RestoreStaple(item: fridge$0.Staple) {
  return call<fridge$0.Staple>("FridgeService", "RestoreStaple", item);
}

export function PlanCookConsumption(entryID: string) {
  return call<fridge$0.ConsumptionPlan>("FridgeService", "PlanCookConsumption", entryID);
}

export function Consume(items: fridge$0.Consumption[] | null) {
  return call<void>("FridgeService", "Consume", items);
}
