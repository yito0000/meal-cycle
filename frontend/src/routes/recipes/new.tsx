import { useQueryClient } from "@tanstack/react-query";
import { RecipeEditForm } from "@/components/recipe/recipe-edit-form";
import { router } from "@/router";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export function RecipeNewPage() {
  const queryClient = useQueryClient();

  return (
    <RecipeEditForm
      mode="new"
      initialRecipe={null}
      onSave={async (input) => {
        // Wails IPC は JSON でシリアライズするので、生成される class インスタンスでも
        // interface を満たすリテラルでもよい。TS モード (task build) では
        // モデルは interface として出力されるため、new せずリテラルで組む。
        const draft: Recipe = {
          id: "",
          name: input.name,
          servings: input.servings,
          tags: input.tags,
          ingredients: input.ingredients,
          ingredientCount: input.ingredients.length,
          steps: input.steps,
          notes: input.notes,
          createdAt: "",
          updatedAt: "",
        };
        const created = await RecipeService.Create(draft);
        await queryClient.invalidateQueries({ queryKey: ["recipes", "list"] });
        router.navigate({ to: "/recipes", search: { highlight: created.id } });
        return created;
      }}
      onCancel={() => router.navigate({ to: "/recipes", search: {} })}
    />
  );
}
