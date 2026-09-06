// 生成 bindings (internal/service/index.ts) と同じ形の re-export。
// E2E ビルドでは vite-plugin-e2e-bindings がこのモジュールを生成物の
// 代わりに解決する (design.md D2)。
import type * as Generated from "../../bindings/github.com/yito0000/meal-cycle/internal/service/index.js";

import * as HealthService from "./healthservice.js";
import * as MealPlanService from "./mealplanservice.js";
import * as RecipeService from "./recipeservice.js";

export type { Health } from "../../bindings/github.com/yito0000/meal-cycle/internal/service/models.js";
export { HealthService, MealPlanService, RecipeService };

// 生成 bindings に無い関数、シグネチャ違い、export 漏れはここで type-check が落ちる。
// 実行時アサーションにしないのは、E2E を回さない通常の開発サイクルでも
// ずれを検出させるため (design.md D5)。
const _typeCheck: typeof Generated = { HealthService, MealPlanService, RecipeService };
void _typeCheck;
