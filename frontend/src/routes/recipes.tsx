import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { router } from "@/router";
import { RecipeService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { RecipeSearchInput } from "../components/recipe/recipe-search-input";
import {
  compareRecipes,
  DEFAULT_RECIPE_SORT,
  type RecipeSortKey,
} from "../components/recipe/recipe-sort";
import { RecipeSortSelect } from "../components/recipe/recipe-sort-select";
import { RecipeTable } from "../components/recipe/recipe-table";

// S02 で保存直後の行をハイライトする時間 (ms)。
// TODO(add-config-service): defaults.RecipeListHighlightMs をフロントへ露出する
// Wails サービス (S09 相当) が入ったら差し替える。
const HIGHLIGHT_MS = 2000;

export function RecipesPage() {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<RecipeSortKey>(DEFAULT_RECIPE_SORT);
  const { highlight } = useSearch({ from: "/recipes" });

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

  // ハイライト値は一定時間後に URL search から自動クリアする。
  useEffect(() => {
    if (!highlight) return;
    const timer = window.setTimeout(() => {
      router.navigate({ to: "/recipes", search: {} });
    }, HIGHLIGHT_MS);
    return () => window.clearTimeout(timer);
  }, [highlight]);

  return (
    <div className="flex h-full flex-col gap-4 p-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("pages.recipes.title")}</h1>
        <Button type="button" onClick={() => router.navigate({ to: "/recipes/new" })}>
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
      {data && filteredSorted.length > 0 && (
        <RecipeTable
          recipes={filteredSorted}
          highlightId={highlight}
          onRowClick={(id) =>
            router.navigate({ to: "/recipes/$recipeId/edit", params: { recipeId: id } })
          }
        />
      )}
    </div>
  );
}
