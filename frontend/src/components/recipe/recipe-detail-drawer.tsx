import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { formatAmount } from "@/lib/format-amount";
import { formatDisplayDate } from "@/lib/format-date";
import type {
  Recipe,
  RecipeIngredient,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { formatDate } from "../mealplan/week-utils";
import { RecipeMarkdown } from "./recipe-markdown";
import { useRecipeDetail } from "./use-recipe-detail";

type Props = {
  recipeId: string | null;
  onClose: () => void;
  onEdit: (id: string) => void;
};

export function RecipeDetailDrawer({ recipeId, onClose, onEdit }: Props) {
  const { t } = useTranslation();
  // 閉じ始めに recipeId が null になっても、閉じ切るまで同じ中身を描く (design.md D5)。
  const [shownId, setShownId] = useState<string | null>(recipeId);
  if (recipeId !== null && recipeId !== shownId) setShownId(recipeId);
  const { data, isLoading, isError } = useRecipeDetail(shownId);

  return (
    <Sheet open={recipeId !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent closeLabel={t("pages.recipes.detail.close")}>
        {isLoading && (
          <>
            <SheetTitle className="sr-only">{t("pages.recipes.detail.loading")}</SheetTitle>
            <p className="p-6 text-sm text-neutral-500">{t("pages.recipes.detail.loading")}</p>
          </>
        )}
        {(isError || (!isLoading && !data)) && (
          <>
            <SheetTitle className="sr-only">{t("pages.recipes.detail.load_error")}</SheetTitle>
            <p className="p-6 text-sm text-red-700">{t("pages.recipes.detail.load_error")}</p>
          </>
        )}
        {data && <RecipeDetailBody recipe={data} onEdit={onEdit} />}
      </SheetContent>
    </Sheet>
  );
}

function RecipeDetailBody({ recipe, onEdit }: { recipe: Recipe; onEdit: (id: string) => void }) {
  const { t } = useTranslation();
  const tags = recipe.tags ?? [];
  const ingredients = recipe.ingredients ?? [];

  return (
    <>
      <SheetHeader>
        <SheetTitle>{recipe.name}</SheetTitle>
        <SheetDescription className="flex flex-wrap gap-x-4 gap-y-1">
          {tags.length > 0 && <span>{tags.join(" / ")}</span>}
          <span>{t("pages.recipes.detail.servings", { count: recipe.servings })}</span>
          <span>
            {t("pages.recipes.detail.updated_at", {
              date: formatDisplayDate(formatDate(new Date(recipe.updatedAt))),
            })}
          </span>
        </SheetDescription>
      </SheetHeader>

      <div className="flex flex-1 flex-col gap-6 overflow-y-auto p-6">
        <Section title={t("pages.recipes.detail.section.ingredients")}>
          {ingredients.length === 0 ? (
            <Empty />
          ) : (
            <ul className="flex flex-col gap-1 text-sm">
              {ingredients.map((ing, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: 材料は ID を持たず同名もありうる。並びは保存順で固定。
                <li key={i} className="flex gap-3">
                  <span className="min-w-32">{ing.name}</span>
                  <span className="text-neutral-700 tabular-nums">
                    {formatIngredientAmount(ing)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <Section title={t("pages.recipes.detail.section.steps")}>
          {recipe.steps.trim() === "" ? <Empty /> : <RecipeMarkdown text={recipe.steps} />}
        </Section>

        <Section title={t("pages.recipes.detail.section.notes")}>
          {recipe.notes.trim() === "" ? <Empty /> : <RecipeMarkdown text={recipe.notes} />}
        </Section>
      </div>

      <SheetFooter>
        <Button type="button" onClick={() => onEdit(recipe.id)}>
          {t("pages.recipes.detail.edit")}
        </Button>
      </SheetFooter>
    </>
  );
}

// 量 0 は「塩 少々」のような量を持たない材料なので、単位だけを出す。
function formatIngredientAmount(ing: RecipeIngredient): string {
  if (ing.amount === 0) return ing.unit;
  return ing.unit ? `${formatAmount(ing.amount)} ${ing.unit}` : formatAmount(ing.amount);
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
      {children}
    </section>
  );
}

function Empty() {
  const { t } = useTranslation();
  return <p className="text-sm text-neutral-400">{t("pages.recipes.detail.empty")}</p>;
}
