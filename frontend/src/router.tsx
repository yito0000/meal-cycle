import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { z } from "zod";
import { RootLayout } from "@/components/layout/root-layout";
import { detailSearchSchema } from "@/components/recipe/detail-origin";
import { RECIPE_SORT_KEYS, type RecipeSortKey } from "@/components/recipe/recipe-sort";
import { shoppingSearchSchema } from "@/components/shopping/shopping-period";
import { FridgePage } from "@/routes/fridge";
import { MealPlanPage } from "@/routes/meal-plan";
import { RecipesPage } from "@/routes/recipes";
import { RecipeDetailPage } from "@/routes/recipes/$recipeId";
import { RecipeEditPage } from "@/routes/recipes/$recipeId.edit";
import { RecipeNewPage } from "@/routes/recipes/new";
import { ShoppingPage } from "@/routes/shopping";

const rootRoute = createRootRoute({ component: RootLayout });

// start は S03 から戻ったときに表示期間を復元するためだけに使う。サイドバー (search 無し) は今週になる (revise-recipe-detail-page D4)。
const mealPlanSearchSchema = z.object({
  start: z.string().optional().catch(undefined),
});

const mealPlanRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: MealPlanPage,
  validateSearch: mealPlanSearchSchema,
});

// S02 は保存直後のハイライト対象を search param で受け取る (design.md D6)。
// q / sort は S03 から戻ったときに検索語と並び順を復元するために持つ (revise-recipe-detail-page D3)。
const recipesSearchSchema = z.object({
  highlight: z.string().optional(),
  q: z.string().optional().catch(undefined),
  sort: z
    .enum(RECIPE_SORT_KEYS as [RecipeSortKey, ...RecipeSortKey[]])
    .optional()
    .catch(undefined),
});

const recipesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recipes",
  component: RecipesPage,
  validateSearch: recipesSearchSchema,
});

const recipeNewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recipes/new",
  component: RecipeNewPage,
});

const recipeDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recipes/$recipeId",
  component: RecipeDetailPage,
  validateSearch: detailSearchSchema,
});

const recipeEditRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recipes/$recipeId/edit",
  component: RecipeEditPage,
  validateSearch: detailSearchSchema,
});

const fridgeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/fridge",
  component: FridgePage,
});

const shoppingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/shopping",
  component: ShoppingPage,
  // S01 の CTA と S07 上の期間切替が from / days を渡す。サイドバー (search 無し) は既定期間になる (design.md D8)。
  validateSearch: shoppingSearchSchema,
});

const routeTree = rootRoute.addChildren([
  mealPlanRoute,
  recipesRoute,
  recipeNewRoute,
  recipeDetailRoute,
  recipeEditRoute,
  fridgeRoute,
  shoppingRoute,
]);

// Wails は webview の URL が固定 (`/`) なので memory history にする。
// ブラウザの戻る/進むは使わないが、router 内 navigate は動く。
export const router = createRouter({
  routeTree,
  history: createMemoryHistory({ initialEntries: ["/"] }),
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
