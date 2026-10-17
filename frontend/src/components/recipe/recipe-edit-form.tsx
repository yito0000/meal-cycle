import { useMutation } from "@tanstack/react-query";
import { useBlocker } from "@tanstack/react-router";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useIngredientSuggestions } from "@/components/ingredient/use-ingredient-suggestions";
import { type BreadcrumbEntry, PageBreadcrumb } from "@/components/layout/breadcrumb";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type {
  Recipe,
  RecipeIngredient,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeIngredientRow } from "./recipe-ingredient-row";
import { RecipeTagsInput } from "./recipe-tags-input";
import { UnsavedChangesDialog } from "./unsaved-changes-dialog";
import { blankFormValues, recipeToFormValues, useRecipeForm } from "./use-recipe-form";

// バリデーション上限 (Go 側 internal/service/recipe.go と揃える。design.md D12)。
const NAME_MAX = 200;
const SERVINGS_MIN = 1;
const SERVINGS_MAX = 99;
const STEPS_MAX = 4000;
const NOTES_MAX = 2000;
const AMOUNT_MAX = 99999;

type SaveInput = {
  name: string;
  servings: number;
  tags: string[];
  ingredients: RecipeIngredient[];
  steps: string;
  notes: string;
};

type Props = {
  mode: "new" | "edit";
  initialRecipe: Recipe | null;
  onSave: (input: SaveInput) => Promise<Recipe>;
  onCancel: () => void;
  breadcrumb: BreadcrumbEntry[];
};

type FieldErrors = {
  name?: string;
  servings?: string;
  steps?: string;
  notes?: string;
  ingredients?: (string | undefined)[];
  ingredientAmounts?: (string | undefined)[];
};

// Wails v3 は型付きエラーコードをバインディングに乗せない (err.message にメッセージ
// 文字列が入るだけ) ので、部分一致で i18n key に対応付ける。
function mapErrorToI18nKey(err: unknown): string {
  const msg = err instanceof Error ? err.message : String(err ?? "");
  if (msg.includes("recipe name empty")) return "pages.recipes.edit.save_error.name_empty";
  if (msg.includes("recipe name too long")) return "pages.recipes.edit.save_error.name_too_long";
  if (msg.includes("servings out of range"))
    return "pages.recipes.edit.save_error.servings_out_of_range";
  if (msg.includes("ingredient name empty"))
    return "pages.recipes.edit.save_error.ingredient_name_required";
  if (msg.includes("ingredient amount out of range"))
    return "pages.recipes.edit.save_error.amount_out_of_range";
  if (msg.includes("steps too long")) return "pages.recipes.edit.save_error.steps_too_long";
  if (msg.includes("notes too long")) return "pages.recipes.edit.save_error.notes_too_long";
  if (msg.includes("tag too long")) return "pages.recipes.edit.save_error.tag_too_long";
  if (msg.includes("recipe not found")) return "pages.recipes.edit.load_error";
  return "pages.recipes.edit.save_error.generic";
}

// Go 側 (最終防衛線) と同じルールを二重で書く。即時フィードバック用。
// 量は数に直す前の文字列で見る。Number("1e400") の Infinity は JSON で null になり、Go 側で量 0 に化けるため。
function validate(values: SaveInput, amountTexts: string[]): FieldErrors {
  const errors: FieldErrors = {};
  const name = values.name.trim();
  if (name === "") errors.name = "pages.recipes.edit.validation.name_required";
  else if ([...name].length > NAME_MAX) errors.name = "pages.recipes.edit.save_error.name_too_long";

  if (
    !Number.isInteger(values.servings) ||
    values.servings < SERVINGS_MIN ||
    values.servings > SERVINGS_MAX
  ) {
    errors.servings = "pages.recipes.edit.save_error.servings_out_of_range";
  }
  if ([...values.steps].length > STEPS_MAX) {
    errors.steps = "pages.recipes.edit.save_error.steps_too_long";
  }
  if ([...values.notes].length > NOTES_MAX) {
    errors.notes = "pages.recipes.edit.save_error.notes_too_long";
  }

  const ingErrors: (string | undefined)[] = [];
  let hasIngErr = false;
  for (const ing of values.ingredients) {
    if (ing.name.trim() === "") {
      ingErrors.push("pages.recipes.edit.save_error.ingredient_name_required");
      hasIngErr = true;
    } else {
      ingErrors.push(undefined);
    }
  }
  if (hasIngErr) errors.ingredients = ingErrors;

  let hasAmountErr = false;
  const amountErrors = amountTexts.map((text) => {
    if (text.trim() === "") return undefined;
    const n = Number(text);
    if (Number.isFinite(n) && n >= 0 && n <= AMOUNT_MAX) return undefined;
    hasAmountErr = true;
    return "pages.recipes.edit.save_error.amount_out_of_range";
  });
  if (hasAmountErr) errors.ingredientAmounts = amountErrors;

  return errors;
}

export function RecipeEditForm({ mode, initialRecipe, onSave, onCancel, breadcrumb }: Props) {
  const { t } = useTranslation();

  const suggestions = useIngredientSuggestions();
  const form = useRecipeForm(initialRecipe ? recipeToFormValues(initialRecipe) : blankFormValues);
  const [saveErrorKey, setSaveErrorKey] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  // 保存後の移動は onSave の中で起き、描画を待たずに通す必要があるので ref で持つ (fix-recipe-form-and-list design.md D2)。
  const leavingAfterSave = useRef(false);

  // キャンセル、パンくず、サイドバーのどれで出ても同じ確認に乗せる (fix-recipe-form-and-list design.md D1)。
  const blocker = useBlocker({
    shouldBlockFn: () => form.isDirty && !leavingAfterSave.current,
    withResolver: true,
    enableBeforeUnload: false,
  });

  const saveMutation = useMutation({
    mutationFn: (input: SaveInput) => onSave(input),
    onMutate: () => {
      leavingAfterSave.current = true;
    },
    onError: (err: unknown) => {
      // 戻さないと、保存に失敗したあとの移動で未保存の確認が出なくなる。
      leavingAfterSave.current = false;
      setSaveErrorKey(mapErrorToI18nKey(err));
    },
  });

  const stepsRemaining = useMemo(
    () => STEPS_MAX - [...form.values.steps].length,
    [form.values.steps],
  );
  const notesRemaining = useMemo(
    () => NOTES_MAX - [...form.values.notes].length,
    [form.values.notes],
  );

  const handleSave = () => {
    setSaveErrorKey(null);
    const input: SaveInput = {
      name: form.values.name.trim(),
      // 空は Number("") が 0 になり範囲外、小数や数字でない値は isInteger で弾かれ、どれも人数のエラーになる。
      servings: Number(form.values.servings),
      tags: form.values.tags,
      ingredients: form.values.ingredients.map((ing) => ({
        name: ing.name,
        amount: ing.amount.trim() === "" ? 0 : Number(ing.amount),
        unit: ing.unit,
      })),
      steps: form.values.steps,
      notes: form.values.notes,
    };
    const errs = validate(
      input,
      form.values.ingredients.map((ing) => ing.amount),
    );
    setFieldErrors(errs);
    if (Object.keys(errs).length > 0) return;
    saveMutation.mutate(input);
  };

  const titleKey =
    mode === "new" ? "pages.recipes.edit.title_new" : "pages.recipes.edit.title_edit";
  const title = t(titleKey, { name: initialRecipe?.name ?? "" });

  return (
    <div className="flex h-full flex-col">
      <div className="px-6 pt-4">
        <PageBreadcrumb items={breadcrumb} />
      </div>
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-neutral-200 bg-white px-6 py-3">
        <h1 className="text-xl font-bold">{title}</h1>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            {t("pages.recipes.edit.cancel")}
          </Button>
          <Button type="button" onClick={handleSave} disabled={saveMutation.isPending}>
            {t("pages.recipes.edit.save")}
          </Button>
        </div>
      </header>

      {saveErrorKey && (
        <div className="border-b border-red-300 bg-red-50 px-6 py-2 text-sm text-red-700">
          {t(saveErrorKey)}
        </div>
      )}

      <div className="grid flex-1 grid-cols-1 gap-6 overflow-auto p-6 lg:grid-cols-2">
        <section className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="recipe-name">{t("pages.recipes.edit.label.name")}</Label>
            <Input
              id="recipe-name"
              value={form.values.name}
              maxLength={NAME_MAX}
              onChange={(e) => form.setName(e.target.value)}
              aria-invalid={fieldErrors.name != null}
            />
            {fieldErrors.name && <p className="text-xs text-red-700">{t(fieldErrors.name)}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="recipe-servings">{t("pages.recipes.edit.label.servings")}</Label>
            <Input
              id="recipe-servings"
              type="number"
              min={SERVINGS_MIN}
              max={SERVINGS_MAX}
              step={1}
              value={form.values.servings}
              onChange={(e) => form.setServings(e.target.value)}
              className="w-24"
              aria-invalid={fieldErrors.servings != null}
            />
            {fieldErrors.servings && (
              <p className="text-xs text-red-700">{t(fieldErrors.servings)}</p>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t("pages.recipes.edit.label.tags")}</Label>
            <RecipeTagsInput value={form.values.tags} onChange={form.setTags} />
          </div>

          <div className="flex flex-col gap-2">
            <Label>{t("pages.recipes.edit.label.ingredients")}</Label>
            {form.values.ingredients.map((row, i) => (
              <RecipeIngredientRow
                // biome-ignore lint/suspicious/noArrayIndexKey: 材料は順序が意味を持つ。
                key={i}
                value={row}
                nameError={fieldErrors.ingredients?.[i] ?? null}
                amountError={fieldErrors.ingredientAmounts?.[i] ?? null}
                nameSuggestions={suggestions.data?.names ?? []}
                unitSuggestions={suggestions.data?.units ?? []}
                onChange={(next) => {
                  const copy = form.values.ingredients.slice();
                  copy[i] = next;
                  form.setIngredients(copy);
                }}
                onRemove={() => {
                  form.setIngredients(form.values.ingredients.filter((_, idx) => idx !== i));
                }}
              />
            ))}
            <Button
              type="button"
              variant="outline"
              onClick={() =>
                form.setIngredients([
                  ...form.values.ingredients,
                  { name: "", amount: "", unit: "" },
                ])
              }
            >
              {t("pages.recipes.edit.add_ingredient")}
            </Button>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="recipe-steps">{t("pages.recipes.edit.label.steps")}</Label>
            <Textarea
              id="recipe-steps"
              value={form.values.steps}
              maxLength={STEPS_MAX}
              onChange={(e) => form.setSteps(e.target.value)}
              rows={12}
              aria-invalid={fieldErrors.steps != null}
            />
            <div className="flex items-center justify-between text-xs text-neutral-500">
              <span>{t("pages.recipes.edit.markdown_hint")}</span>
              <span className={stepsRemaining < 0 ? "text-red-700" : ""}>
                {t("pages.recipes.edit.chars_remaining", { count: stepsRemaining })}
              </span>
            </div>
            {fieldErrors.steps && <p className="text-xs text-red-700">{t(fieldErrors.steps)}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="recipe-notes">{t("pages.recipes.edit.label.notes")}</Label>
            <Textarea
              id="recipe-notes"
              value={form.values.notes}
              maxLength={NOTES_MAX}
              onChange={(e) => form.setNotes(e.target.value)}
              rows={6}
              aria-invalid={fieldErrors.notes != null}
            />
            <div className="text-right text-xs text-neutral-500">
              <span className={notesRemaining < 0 ? "text-red-700" : ""}>
                {t("pages.recipes.edit.chars_remaining", { count: notesRemaining })}
              </span>
            </div>
            {fieldErrors.notes && <p className="text-xs text-red-700">{t(fieldErrors.notes)}</p>}
          </div>
        </section>
      </div>

      <UnsavedChangesDialog
        open={blocker.status === "blocked"}
        onDiscard={() => blocker.proceed?.()}
        onKeepEditing={() => blocker.reset?.()}
      />
    </div>
  );
}
