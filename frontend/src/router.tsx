import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { RootLayout } from "@/components/layout/root-layout";
import { FridgePage } from "@/routes/fridge";
import { MealPlanPage } from "@/routes/meal-plan";
import { RecipesPage } from "@/routes/recipes";
import { ShoppingPage } from "@/routes/shopping";

const rootRoute = createRootRoute({ component: RootLayout });

const mealPlanRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: MealPlanPage,
});

const recipesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recipes",
  component: RecipesPage,
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
});

const routeTree = rootRoute.addChildren([mealPlanRoute, recipesRoute, fridgeRoute, shoppingRoute]);

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
