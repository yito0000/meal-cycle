import type * as recipe$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models.js";
import { call } from "./call.js";

export function Create(input: recipe$0.Recipe) {
  return call<recipe$0.Recipe>("RecipeService", "Create", input);
}

export function Delete(id: string) {
  return call<void>("RecipeService", "Delete", id);
}

export function GetByID(id: string) {
  return call<recipe$0.Recipe>("RecipeService", "GetByID", id);
}

export function Import(inputs: recipe$0.Recipe[] | null) {
  return call<recipe$0.Recipe[]>("RecipeService", "Import", inputs);
}

export function List() {
  return call<recipe$0.Recipe[]>("RecipeService", "List");
}

export function PreviewImport(files: recipe$0.ImportFile[] | null) {
  return call<recipe$0.ImportCandidate[]>("RecipeService", "PreviewImport", files);
}

export function Update(id: string, input: recipe$0.Recipe) {
  return call<recipe$0.Recipe>("RecipeService", "Update", id, input);
}
