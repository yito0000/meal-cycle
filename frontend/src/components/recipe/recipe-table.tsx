import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatTimestampDate } from "@/lib/format-date";
import { cn } from "@/lib/utils";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

type Props = {
  recipes: Recipe[];
  onRowClick?: (id: string) => void;
  highlightId?: string;
};

export function RecipeTable({ recipes, onRowClick, highlightId }: Props) {
  const { t } = useTranslation();
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[35%]">{t("pages.recipes.column.name")}</TableHead>
          <TableHead className="w-[45%]">{t("pages.recipes.column.tags")}</TableHead>
          <TableHead className="w-[20%]">{t("pages.recipes.column.updated_at")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {recipes.map((r) => (
          <TableRow
            key={r.id}
            className={cn(onRowClick && "cursor-pointer", highlightId === r.id && "bg-accent")}
            onClick={() => onRowClick?.(r.id)}
          >
            <TableCell className="font-medium">
              {onRowClick ? (
                <button
                  type="button"
                  className="cursor-pointer rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                  onClick={(e) => {
                    // 行の onClick と二重に開かないようにする。
                    e.stopPropagation();
                    onRowClick(r.id);
                  }}
                >
                  {r.name}
                </button>
              ) : (
                r.name
              )}
            </TableCell>
            <TableCell className="text-neutral-700">{(r.tags ?? []).join(" / ")}</TableCell>
            <TableCell className="text-neutral-700 tabular-nums">
              {formatTimestampDate(r.updatedAt)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
