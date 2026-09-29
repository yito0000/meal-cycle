import { z } from "zod";
import { RECIPE_SORT_KEYS, type RecipeSortKey } from "./recipe-sort";

// S03 / S04 をどの画面から開いたかと、戻ったときに復元するその画面の状態 (design.md D2)。
export type DetailOrigin =
  | { to: "recipes"; q?: string; sort?: RecipeSortKey }
  | { to: "mealplan"; start?: string };

const recipesOriginSchema = z.object({
  to: z.literal("recipes"),
  q: z.string().optional().catch(undefined),
  sort: z
    .enum(RECIPE_SORT_KEYS as [RecipeSortKey, ...RecipeSortKey[]])
    .optional()
    .catch(undefined),
});

const mealplanOriginSchema = z.object({
  to: z.literal("mealplan"),
  start: z.string().optional().catch(undefined),
});

// 壊れた back でパンくずと戻り先が決まらなくならないよう、S02 に倒す。
export const detailOriginSchema = z
  .discriminatedUnion("to", [recipesOriginSchema, mealplanOriginSchema])
  .catch({ to: "recipes" });

export const detailSearchSchema = z.object({
  back: detailOriginSchema.optional().catch(undefined),
});

export const DEFAULT_DETAIL_ORIGIN: DetailOrigin = { to: "recipes" };

export function originNavigateOptions(origin: DetailOrigin) {
  if (origin.to === "mealplan") {
    return { to: "/", search: { start: origin.start } } as const;
  }
  return { to: "/recipes", search: { q: origin.q, sort: origin.sort } } as const;
}

export function originLabelKey(origin: DetailOrigin): string {
  return origin.to === "mealplan" ? "nav.meal_plan" : "nav.recipes";
}
