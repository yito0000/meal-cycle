import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { RecipeService } from "../../bindings/github.com/yito0000/meal-cycle";
import { RecipeSearchInput } from "../components/recipe/recipe-search-input";
import {
  compareRecipes,
  DEFAULT_RECIPE_SORT,
  type RecipeSortKey,
} from "../components/recipe/recipe-sort";
import { RecipeSortSelect } from "../components/recipe/recipe-sort-select";
import { RecipeTable } from "../components/recipe/recipe-table";

export function RecipesPage() {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<RecipeSortKey>(DEFAULT_RECIPE_SORT);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["recipes", "list"],
    queryFn: () => RecipeService.List(),
  });

  const filteredSorted = useMemo(() => {
    if (!data) return [];
    const q = search.trim().toLowerCase();
    const filtered = q ? data.filter((r) => r.name.toLowerCase().includes(q)) : data;
    return [...filtered].sort((a, b) => compareRecipes(a, b, sortKey));
  }, [data, search, sortKey]);

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("pages.recipes.title")}</h1>
        <Button type="button" onClick={() => {}}>
          {t("pages.recipes.new_button")}
        </Button>
      </header>

      <div className="flex items-center justify-between gap-4">
        <RecipeSearchInput value={search} onChange={setSearch} />
        <RecipeSortSelect value={sortKey} onChange={setSortKey} />
      </div>

      {isLoading && <p className="text-sm text-neutral-500">{t("pages.recipes.loading")}</p>}
      {isError && <p className="text-sm text-red-700">{t("pages.recipes.load_error")}</p>}
      {data && data.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.recipes.empty.no_recipes")}</p>
      )}
      {data && data.length > 0 && filteredSorted.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.recipes.empty.no_match")}</p>
      )}
      {data && filteredSorted.length > 0 && <RecipeTable recipes={filteredSorted} />}
    </div>
  );
}
