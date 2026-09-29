import { useParams, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { type BreadcrumbEntry, PageBreadcrumb } from "@/components/layout/breadcrumb";
import {
  DEFAULT_DETAIL_ORIGIN,
  originLabelKey,
  originNavigateOptions,
} from "@/components/recipe/detail-origin";
import { RecipeDetailBody } from "@/components/recipe/recipe-detail";
import { useRecipeDetail } from "@/components/recipe/use-recipe-detail";
import { router } from "@/router";

export function RecipeDetailPage() {
  const { t } = useTranslation();
  const { recipeId } = useParams({ from: "/recipes/$recipeId" });
  const { back } = useSearch({ from: "/recipes/$recipeId" });
  const origin = back ?? DEFAULT_DETAIL_ORIGIN;
  const { data, isLoading, isError } = useRecipeDetail(recipeId);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const goBack = () => router.navigate(originNavigateOptions(origin));
  const goBackRef = useRef(goBack);
  goBackRef.current = goBack;
  const confirmOpenRef = useRef(confirmOpen);
  confirmOpenRef.current = confirmOpen;

  // capture で拾い、Radix が確認ダイアログを閉じる前の開閉状態で判定する (revise-recipe-detail-page D5)。
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Escape" || confirmOpenRef.current) return;
      goBackRef.current();
    };
    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, []);

  const crumbs: BreadcrumbEntry[] = [{ label: t(originLabelKey(origin)), onSelect: goBack }];
  if (data) crumbs.push({ label: data.name });

  return (
    <div className="flex flex-col gap-6 p-6">
      <PageBreadcrumb items={crumbs} />
      {isLoading && <p className="text-sm text-neutral-500">{t("pages.recipes.detail.loading")}</p>}
      {(isError || (!isLoading && !data)) && (
        <p className="text-sm text-red-700">{t("pages.recipes.detail.load_error")}</p>
      )}
      {data && (
        <RecipeDetailBody
          recipe={data}
          onEdit={(id) =>
            router.navigate({
              to: "/recipes/$recipeId/edit",
              params: { recipeId: id },
              search: { back: origin },
            })
          }
          onDeleted={goBack}
          confirmOpen={confirmOpen}
          onConfirmOpenChange={setConfirmOpen}
        />
      )}
    </div>
  );
}
