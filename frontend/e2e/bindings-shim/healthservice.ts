import type * as $models from "../../bindings/github.com/yito0000/meal-cycle/internal/service/models.js";
import { call } from "./call.js";

export function Ping() {
  return call<$models.Health>("HealthService", "Ping");
}
