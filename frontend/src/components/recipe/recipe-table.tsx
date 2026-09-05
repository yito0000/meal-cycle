import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
            <TableCell className="font-medium">{r.name}</TableCell>
            <TableCell className="text-neutral-700">{(r.tags ?? []).join(" / ")}</TableCell>
            <TableCell className="text-neutral-700 tabular-nums">{r.updatedAt}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
