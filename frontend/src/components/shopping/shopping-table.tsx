import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatAmount } from "@/lib/format-amount";
import { cn } from "@/lib/utils";
import {
  type Item,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { StockStatusBadge } from "./stock-status-badge";

type Props = {
  items: (Item & { recipes: string[] })[];
};

export function ShoppingTable({ items }: Props) {
  const { t } = useTranslation();
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[28%]">{t("pages.shopping.column.name")}</TableHead>
          <TableHead className="w-[16%]">{t("pages.shopping.column.amount")}</TableHead>
          <TableHead className="w-[18%]">{t("pages.shopping.column.stock")}</TableHead>
          <TableHead className="w-[38%]">{t("pages.shopping.column.usage")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => {
          const covered = item.stockStatus === StockStatus.StockCovered;
          return (
            <TableRow
              key={`${item.name}\u0000${item.unit}`}
              data-stock-status={item.stockStatus}
              className={cn(covered && "text-neutral-400")}
            >
              <TableCell>{item.name}</TableCell>
              <TableCell>
                {item.amount === 0
                  ? t("pages.shopping.amount_unspecified")
                  : `${formatAmount(item.amount)} ${item.unit}`.trim()}
              </TableCell>
              <TableCell>
                <StockStatusBadge status={item.stockStatus} />
              </TableCell>
              <TableCell>{item.recipes.join(", ")}</TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
