import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { router } from "@/router";
import type { ImportFile } from "../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeService } from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { RecipeExportButton } from "../components/recipe/recipe-export-button";
import { collectTags, filterRecipes } from "../components/recipe/recipe-filter";
import { RecipeImportButton } from "../components/recipe/recipe-import-button";
import { RecipeImportDialog } from "../components/recipe/recipe-import-dialog";
import { RecipeSearchInput } from "../components/recipe/recipe-search-input";
import { compareRecipes, DEFAULT_RECIPE_SORT } from "../components/recipe/recipe-sort";
import { RecipeSortSelect } from "../components/recipe/recipe-sort-select";
import { RecipeTable } from "../components/recipe/recipe-table";
import { RecipeTagFilter } from "../components/recipe/recipe-tag-filter";

// S02 保存直後のハイライト時間 (ms)。S09 で ConfigService が Wails 経由で
// defaults.RecipeListHighlightMs を返すようになったら差し替える。
const HIGHLIGHT_MS = 2000;

export function RecipesPage() {
  const { t } = useTranslation();
  const { highlight, q, sort, tags } = useSearch({ from: "/recipes" });
  const sortKey = sort ?? DEFAULT_RECIPE_SORT;
  // 入力欄を URL の値で直接制御すると、ルータの更新待ちの間に IME の変換中の文字が崩れうる (revise-recipe-detail-page D3)。
  const [search, setSearch] = useState(q ?? "");
  const [importRequest, setImportRequest] = useState<{ id: number; files: ImportFile[] } | null>(
    null,
  );
  // 閉じても数え続け、読み直すたびに PreviewImport のクエリを別物にする。
  const importSeq = useRef(0);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["recipes", "list"],
    queryFn: () => RecipeService.List(),
  });

  const allTags = useMemo(() => (data ? collectTags(data) : []), [data]);
  // 選んでいたタグを持つレシピがすべて消えたら、そのタグは選択から外す (add-recipe-tag-filter design.md D3)。
  const selectedTags = useMemo(
    () => (tags ?? []).filter((tag) => !data || allTags.includes(tag)),
    [tags, data, allTags],
  );

  const filteredSorted = useMemo(() => {
    if (!data) return [];
    const filtered = filterRecipes(data, { query: search, tags: selectedTags });
    return filtered.sort((a, b) => compareRecipes(a, b, sortKey));
  }, [data, search, selectedTags, sortKey]);

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
        <div className="flex items-center gap-2">
          <RecipeExportButton disabled={!data || data.length === 0} />
          <RecipeImportButton
            onRead={(files) => {
              importSeq.current += 1;
              setImportRequest({ id: importSeq.current, files });
            }}
          />
          <Button type="button" onClick={() => router.navigate({ to: "/recipes/new" })}>
            {t("pages.recipes.new_button")}
          </Button>
        </div>
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

      <RecipeTagFilter
        tags={allTags}
        selected={selectedTags}
        onChange={(next) =>
          router.navigate({
            to: "/recipes",
            search: (prev) => ({ ...prev, tags: next.length === 0 ? undefined : next }),
            replace: true,
          })
        }
      />

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
              search: {
                back: {
                  to: "recipes",
                  q,
                  sort,
                  tags: selectedTags.length === 0 ? undefined : selectedTags,
                },
              },
            })
          }
        />
      )}

      {importRequest && (
        <RecipeImportDialog
          key={importRequest.id}
          requestId={importRequest.id}
          files={importRequest.files}
          onClose={() => setImportRequest(null)}
        />
      )}
    </div>
  );
}
