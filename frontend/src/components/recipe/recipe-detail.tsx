import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { formatDisplayDate } from "@/lib/format-date";
import { formatQuantity } from "@/lib/format-quantity";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { formatDate } from "../mealplan/week-utils";
import { DeleteRecipeDialog } from "./delete-recipe-dialog";
import { RecipeMarkdown } from "./recipe-markdown";

type Props = {
  recipe: Recipe;
  onEdit: (id: string) => void;
  onDeleted: () => void;
  // Esc で戻るかの判定に使うため、確認ダイアログの開閉は親が持つ (revise-recipe-detail-page D5)。
  confirmOpen: boolean;
  onConfirmOpenChange: (open: boolean) => void;
};

export function RecipeDetailBody({
  recipe,
  onEdit,
  onDeleted,
  confirmOpen,
  onConfirmOpenChange,
}: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const tags = recipe.tags ?? [];
  const ingredients = recipe.ingredients ?? [];
  const archived = recipe.archivedAt !== "";

  const deleteMutation = useMutation({
    mutationFn: () => RecipeService.Delete(recipe.id),
    onSuccess: () => {
      onConfirmOpenChange(false);
      queryClient.invalidateQueries({ queryKey: ["recipes", "list"] });
      // アンマウント前に削除済みの表示へ一瞬切り替わらないよう、今は再取得しない (revise-recipe-detail-page D11)。
      queryClient.invalidateQueries({
        queryKey: ["recipes", "detail", recipe.id],
        refetchType: "none",
      });
      toast(t("pages.recipes.delete_confirm.done", { name: recipe.name }));
      onDeleted();
    },
  });

  return (
    <>
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold">{recipe.name}</h1>
          <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-neutral-600">
            {tags.length > 0 && <span>{tags.join(" / ")}</span>}
            <span>{t("pages.recipes.detail.servings", { count: recipe.servings })}</span>
            <span>
              {t("pages.recipes.detail.updated_at", {
                date: formatDisplayDate(formatDate(new Date(recipe.updatedAt))),
              })}
            </span>
            {archived && (
              <span className="font-semibold text-red-700">
                {t("pages.recipes.detail.archived")}
              </span>
            )}
          </p>
        </div>
        {!archived && (
          <div className="flex items-center gap-2">
            <Button type="button" onClick={() => onEdit(recipe.id)}>
              {t("pages.recipes.detail.edit")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                deleteMutation.reset();
                onConfirmOpenChange(true);
              }}
            >
              {t("pages.recipes.detail.delete")}
            </Button>
          </div>
        )}
      </header>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-[20rem_1fr]">
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
                    {formatQuantity(ing.amount, ing.unit)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <div className="flex max-w-3xl flex-col gap-6">
          <Section title={t("pages.recipes.detail.section.steps")}>
            {recipe.steps.trim() === "" ? <Empty /> : <RecipeMarkdown text={recipe.steps} />}
          </Section>

          <Section title={t("pages.recipes.detail.section.notes")}>
            {recipe.notes.trim() === "" ? <Empty /> : <RecipeMarkdown text={recipe.notes} />}
          </Section>
        </div>
      </div>

      <DeleteRecipeDialog
        open={confirmOpen}
        recipeName={recipe.name}
        pending={deleteMutation.isPending}
        error={deleteMutation.isError}
        onConfirm={() => deleteMutation.mutate()}
        onCancel={() => onConfirmOpenChange(false)}
      />
    </>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-neutral-900">{title}</h2>
      {children}
    </section>
  );
}

function Empty() {
  const { t } = useTranslation();
  return <p className="text-sm text-neutral-400">{t("pages.recipes.detail.empty")}</p>;
}
