import { useTranslation } from "react-i18next";
import type { ImportCandidate } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeContentSections } from "./recipe-detail";
import { importIssueMessage } from "./recipe-import-list";

export function RecipeImportPreview({ candidate }: { candidate: ImportCandidate }) {
  const { t } = useTranslation();

  if (candidate.issue) {
    return (
      <div className="flex flex-col gap-1 rounded-md border border-red-200 bg-red-50 p-4 text-sm">
        <p className="font-semibold text-red-800">{t("pages.recipes.import.issue_title")}</p>
        <p className="text-neutral-700">{candidate.fileName}</p>
        <p className="text-red-700">{importIssueMessage(t, candidate)}</p>
      </div>
    );
  }

  const { recipe } = candidate;
  const tags = recipe.tags ?? [];
  return (
    <div className="flex min-h-0 flex-col gap-4 overflow-y-auto">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-bold">{recipe.name}</h2>
        <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-neutral-600">
          {tags.length > 0 && <span>{tags.join(" / ")}</span>}
          <span>{t("pages.recipes.detail.servings", { count: recipe.servings })}</span>
          <span>{candidate.fileName}</span>
        </p>
      </div>
      {candidate.duplicate && (
        <p className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
          {t("pages.recipes.import.duplicate_warning")}
        </p>
      )}
      <RecipeContentSections recipe={recipe} />
    </div>
  );
}
