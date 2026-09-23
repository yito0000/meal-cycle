// 生成 bindings (internal/service/index.ts) と同じ形の re-export。
// E2E ビルドでは vite-plugin-e2e-bindings がこのモジュールを生成物の
// 代わりに解決する (design.md D2)。
import type * as Generated from "../../bindings/github.com/yito0000/meal-cycle/internal/service/index.js";

import * as FridgeService from "./fridgeservice.js";
import * as HealthService from "./healthservice.js";
import * as MealPlanService from "./mealplanservice.js";
import * as RecipeService from "./recipeservice.js";

export type { Health } from "../../bindings/github.com/yito0000/meal-cycle/internal/service/models.js";
export { FridgeService, HealthService, MealPlanService, RecipeService };

// 生成 bindings に無い関数、シグネチャ違い、export 漏れはここで type-check が落ちる。
// wails3 の `-ts -i` (Taskfile 側) は Health を interface 型 (値なし) で吐くため、
// 値の一致は runtime バインディング (FridgeService / HealthService / MealPlanService / RecipeService) だけを
// 突き合わせる。型 export の網羅は `export type` で担保する。
type GeneratedRuntime = Omit<typeof Generated, "Health">;
const _typeCheck: GeneratedRuntime = {
  FridgeService,
  HealthService,
  MealPlanService,
  RecipeService,
};
void _typeCheck;
