import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { router } from "@/router";
import { RecipeService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { RecipeSearchInput } from "../components/recipe/recipe-search-input";
import { compareRecipes, DEFAULT_RECIPE_SORT } from "../components/recipe/recipe-sort";
import { RecipeSortSelect } from "../components/recipe/recipe-sort-select";
import { RecipeTable } from "../components/recipe/recipe-table";

// S02 保存直後のハイライト時間 (ms)。S09 で ConfigService が Wails 経由で
// defaults.RecipeListHighlightMs を返すようになったら差し替える。
const HIGHLIGHT_MS = 2000;

export function RecipesPage() {
  const { t } = useTranslation();
  const { highlight, q, sort } = useSearch({ from: "/recipes" });
  const sortKey = sort ?? DEFAULT_RECIPE_SORT;
  // 入力欄を URL の値で直接制御すると、ルータの更新待ちの間に IME の変換中の文字が崩れうる (revise-recipe-detail-page D3)。
  const [search, setSearch] = useState(q ?? "");

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

  // ハイライト値は一定時間後に URL search から自動クリアする。q / sort は残す (design.md D3)。
  useEffect(() => {
    if (!highlight) return;
    const timer = window.setTimeout(() => {
      router.navigate({ to: "/recipes", search: (prev) => ({ ...prev, highlight: undefined }) });
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
        <RecipeSearchInput
          value={search}
          onChange={(next) => {
            setSearch(next);
            router.navigate({
              to: "/recipes",
              search: (prev) => ({ ...prev, q: next === "" ? undefined : next }),
              replace: true,
            });
          }}
        />
        <RecipeSortSelect
          value={sortKey}
          onChange={(next) =>
            router.navigate({
              to: "/recipes",
              search: (prev) => ({ ...prev, sort: next }),
              replace: true,
            })
          }
        />
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
            router.navigate({
              to: "/recipes/$recipeId",
              params: { recipeId: id },
              search: { back: { to: "recipes", q, sort } },
            })
          }
        />
      )}
    </div>
  );
}
