import { deepEqual } from "fast-equals";
import { useCallback, useMemo, useState } from "react";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

// S04 フォームが扱うフィールドの projection。Repository 側で決まる値
// (ID / CreatedAt / UpdatedAt / IngredientCount) はフォーム状態に持たない。
// 人数と量は入力途中の表記 (空、「1.」) を保つため文字列で持ち、保存時に数値へ直す (fix-recipe-form-and-list design.md D3)。
export type IngredientFormRow = { name: string; amount: string; unit: string };

export type RecipeFormValues = {
  name: string;
  servings: string;
  tags: string[];
  ingredients: IngredientFormRow[];
  steps: string;
  notes: string;
};

export const blankFormValues: RecipeFormValues = {
  name: "",
  servings: "1",
  tags: [],
  ingredients: [],
  steps: "",
  notes: "",
};

// Wails の JSON 契約で tags / ingredients に null が来ることがあるので空扱いにする。
export function recipeToFormValues(r: Recipe): RecipeFormValues {
  return {
    name: r.name ?? "",
    servings: String(r.servings ?? 1),
    tags: r.tags ?? [],
    // 量 0 は「量を持たない材料」なので、欄には 0 を出さず空にする。
    ingredients: (r.ingredients ?? []).map((ing) => ({
      name: ing.name,
      amount: ing.amount === 0 ? "" : String(ing.amount),
      unit: ing.unit,
    })),
    steps: r.steps ?? "",
    notes: r.notes ?? "",
  };
}

// 初期値からの deep 比較で dirty を判定する (design.md D6)。
export function useRecipeForm(initial: RecipeFormValues) {
  const [initialValues, setInitialValues] = useState<RecipeFormValues>(initial);
  const [values, setValues] = useState<RecipeFormValues>(initial);

  const isDirty = useMemo(() => !deepEqual(values, initialValues), [values, initialValues]);

  const reset = useCallback((next: RecipeFormValues) => {
    setInitialValues(next);
    setValues(next);
  }, []);

  const setName = useCallback((name: string) => setValues((v) => ({ ...v, name })), []);
  const setServings = useCallback((servings: string) => setValues((v) => ({ ...v, servings })), []);
  const setTags = useCallback((tags: string[]) => setValues((v) => ({ ...v, tags })), []);
  const setIngredients = useCallback(
    (ingredients: IngredientFormRow[]) => setValues((v) => ({ ...v, ingredients })),
    [],
  );
  const setSteps = useCallback((steps: string) => setValues((v) => ({ ...v, steps })), []);
  const setNotes = useCallback((notes: string) => setValues((v) => ({ ...v, notes })), []);

  return {
    values,
    isDirty,
    reset,
    setName,
    setServings,
    setTags,
    setIngredients,
    setSteps,
    setNotes,
  };
}
