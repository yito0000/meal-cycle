import type * as settings$0 from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/settings/models.js";
import { call } from "./call.js";

export function Get() {
  return call<settings$0.Values>("SettingsService", "Get");
}

export function Reset() {
  return call<settings$0.Values>("SettingsService", "Reset");
}

export function Save(values: settings$0.Values) {
  return call<settings$0.Values>("SettingsService", "Save", values);
}
