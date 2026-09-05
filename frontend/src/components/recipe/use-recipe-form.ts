import { deepEqual } from "fast-equals";
import { useCallback, useMemo, useState } from "react";
import type {
  Recipe,
  RecipeIngredient,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

// RecipeFormValues は S04 のフォームが取り扱うフィールドだけを抜き出した shape。
// Recipe の projection のうち、ID / CreatedAt / UpdatedAt / IngredientCount は
// Repository 側で決まる値なのでフォーム状態には持たない。Steps は Markdown 記法の
// 単一テキスト (design.md D10 参照)。
export type RecipeFormValues = {
  name: string;
  servings: number;
  tags: string[];
  ingredients: RecipeIngredient[];
  steps: string;
  notes: string;
};

// blankFormValues は新規登録モードでの初期値。
export const blankFormValues: RecipeFormValues = {
  name: "",
  servings: 1,
  tags: [],
  ingredients: [],
  steps: "",
  notes: "",
};

// recipeToFormValues は既存 Recipe を編集モードの初期値に変換する。
// null-safe: バインディング上 tags/ingredients は配列だが未定義や null が来ても
// 空扱いにする (Wails の JSON 契約で null が来ることがある)。
export function recipeToFormValues(r: Recipe): RecipeFormValues {
  return {
    name: r.name ?? "",
    servings: r.servings ?? 1,
    tags: r.tags ?? [],
    ingredients: r.ingredients ?? [],
    steps: r.steps ?? "",
    notes: r.notes ?? "",
  };
}

// useRecipeForm はフォーム状態と dirty 判定をひとまとめにするフック。
// 初期値からの deep 比較で dirty を提供する (design.md D6)。
export function useRecipeForm(initial: RecipeFormValues) {
  const [initialValues, setInitialValues] = useState<RecipeFormValues>(initial);
  const [values, setValues] = useState<RecipeFormValues>(initial);

  const isDirty = useMemo(() => !deepEqual(values, initialValues), [values, initialValues]);

  const reset = useCallback((next: RecipeFormValues) => {
    setInitialValues(next);
    setValues(next);
  }, []);

  const setName = useCallback((name: string) => setValues((v) => ({ ...v, name })), []);
  const setServings = useCallback((servings: number) => setValues((v) => ({ ...v, servings })), []);
  const setTags = useCallback((tags: string[]) => setValues((v) => ({ ...v, tags })), []);
  const setIngredients = useCallback(
    (ingredients: RecipeIngredient[]) => setValues((v) => ({ ...v, ingredients })),
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
