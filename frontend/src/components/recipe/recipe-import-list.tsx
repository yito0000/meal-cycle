import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { ImportCandidate } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

export function importIssueMessage(t: TFunction, c: ImportCandidate): string {
  return t(`pages.recipes.import.issue.${c.issue}`, { arg: c.issueArg });
}

type Props = {
  candidates: ImportCandidate[];
  selected: Set<string>;
  onSelectedChange: (next: Set<string>) => void;
};

export function RecipeImportList({ candidates, selected, onSelectedChange }: Props) {
  const { t } = useTranslation();

  const toggle = (fileName: string) => {
    const next = new Set(selected);
    if (next.has(fileName)) next.delete(fileName);
    else next.add(fileName);
    onSelectedChange(next);
  };

  return (
    <div className="flex min-h-0 flex-col gap-2">
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() =>
            onSelectedChange(new Set(candidates.filter((c) => !c.issue).map((c) => c.fileName)))
          }
        >
          {t("pages.recipes.import.select_all")}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onSelectedChange(new Set())}
        >
          {t("pages.recipes.import.deselect_all")}
        </Button>
      </div>
      <div className="min-h-0 overflow-y-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10" />
              <TableHead>{t("pages.recipes.import.column.file_name")}</TableHead>
              <TableHead>{t("pages.recipes.import.column.recipe_name")}</TableHead>
              <TableHead className="w-20">
                {t("pages.recipes.import.column.ingredient_count")}
              </TableHead>
              <TableHead>{t("pages.recipes.import.column.status")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {candidates.map((c) => {
              const hasIssue = !!c.issue;
              return (
                <TableRow
                  key={c.fileName}
                  data-testid="recipe-import-row"
                  className={cn(hasIssue && "text-neutral-400")}
                >
                  <TableCell>
                    <input
                      type="checkbox"
                      checked={selected.has(c.fileName)}
                      disabled={hasIssue}
                      onChange={() => toggle(c.fileName)}
                      aria-label={t("pages.recipes.import.select_row", { name: c.fileName })}
                      className="h-4 w-4 align-middle"
                    />
                  </TableCell>
                  <TableCell className="break-all">{c.fileName}</TableCell>
                  <TableCell>{c.recipe.name}</TableCell>
                  <TableCell className="tabular-nums">
                    {hasIssue && c.recipe.name === "" ? "" : c.recipe.ingredientCount}
                  </TableCell>
                  <TableCell>
                    {hasIssue ? (
                      <span className="text-red-700">{importIssueMessage(t, c)}</span>
                    ) : c.duplicate ? (
                      <span className="text-amber-700">
                        {t("pages.recipes.import.status.duplicate")}
                      </span>
                    ) : (
                      t("pages.recipes.import.status.ready")
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
