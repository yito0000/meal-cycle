import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { RecipeEditForm } from "@/components/recipe/recipe-edit-form";
import { useRecipeDetail } from "@/components/recipe/use-recipe-detail";
import { router } from "@/router";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export function RecipeEditPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { recipeId } = useParams({ from: "/recipes/$recipeId/edit" });

  const { data, isLoading, isError } = useRecipeDetail(recipeId);

  if (isLoading) {
    return <p className="p-6 text-sm text-neutral-500">{t("pages.recipes.edit.loading")}</p>;
  }
  if (isError || !data) {
    return (
      <div className="flex flex-col gap-4 p-6 text-sm">
        <p className="text-red-700">{t("pages.recipes.edit.load_error")}</p>
        <button
          type="button"
          className="self-start text-blue-700 underline"
          onClick={() => router.navigate({ to: "/recipes", search: {} })}
        >
          {t("pages.recipes.edit.back_to_list")}
        </button>
      </div>
    );
  }

  return (
    <RecipeEditForm
      mode="edit"
      initialRecipe={data}
      onSave={async (input) => {
        // TS モードバインディング (task build) では Recipe が interface なので、
        // `new Recipe(...)` は使えない。Wails IPC は JSON なのでリテラルで足りる。
        const draft: Recipe = {
          id: recipeId,
          name: input.name,
          servings: input.servings,
          tags: input.tags,
          ingredients: input.ingredients,
          ingredientCount: input.ingredients.length,
          steps: input.steps,
          notes: input.notes,
          createdAt: data.createdAt,
          updatedAt: data.updatedAt,
          archivedAt: "",
        };
        const updated = await RecipeService.Update(recipeId, draft);
        await queryClient.invalidateQueries({ queryKey: ["recipes", "list"] });
        await queryClient.invalidateQueries({ queryKey: ["recipes", "detail", recipeId] });
        router.navigate({ to: "/recipes", search: { highlight: updated.id } });
        return updated;
      }}
      onCancel={() => router.navigate({ to: "/recipes", search: { detail: recipeId } })}
    />
  );
}
