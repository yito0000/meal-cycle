import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatQuantity } from "@/lib/format-quantity";
import { cn } from "@/lib/utils";
import {
  type Item,
  StockStatus,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { itemKey } from "./purchase-form";
import { StockStatusBadge } from "./stock-status-badge";

type Props = {
  items: (Item & { recipes: string[] })[];
  checked: Set<string>;
  onCheckedChange: (next: Set<string>) => void;
};

export function ShoppingTable({ items, checked, onCheckedChange }: Props) {
  const { t } = useTranslation();
  const checkedCount = items.filter((it) => checked.has(itemKey(it))).length;
  const allChecked = items.length > 0 && checkedCount === items.length;
  const headerRef = useRef<HTMLInputElement>(null);

  // indeterminate は HTML 属性が無く DOM プロパティでしか設定できない (design.md D4)。
  useEffect(() => {
    if (headerRef.current) headerRef.current.indeterminate = checkedCount > 0 && !allChecked;
  }, [checkedCount, allChecked]);

  // items は表示中の行だけなので、置き換えではなく差分で更新して隠れた行のチェックを残す (add-shopping-stock-filter D3)。
  const toggleAll = () => {
    const next = new Set(checked);
    for (const it of items) {
      if (allChecked) next.delete(itemKey(it));
      else next.add(itemKey(it));
    }
    onCheckedChange(next);
  };

  const toggle = (key: string) => {
    const next = new Set(checked);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    onCheckedChange(next);
  };

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-10">
            <input
              ref={headerRef}
              type="checkbox"
              checked={allChecked}
              onChange={toggleAll}
              aria-label={t("pages.shopping.purchase.check_all")}
              className="h-4 w-4 align-middle"
            />
          </TableHead>
          <TableHead className="w-[26%]">{t("pages.shopping.column.name")}</TableHead>
          <TableHead className="w-[15%]">{t("pages.shopping.column.amount")}</TableHead>
          <TableHead className="w-[17%]">{t("pages.shopping.column.stock")}</TableHead>
          <TableHead className="w-[36%]">{t("pages.shopping.column.usage")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => {
          const key = itemKey(item);
          const covered = item.stockStatus === StockStatus.StockCovered;
          return (
            <TableRow
              key={key}
              data-stock-status={item.stockStatus}
              className={cn(covered && "text-neutral-400")}
            >
              <TableCell>
                <input
                  type="checkbox"
                  checked={checked.has(key)}
                  onChange={() => toggle(key)}
                  aria-label={t("pages.shopping.purchase.check_row", { name: item.name })}
                  className="h-4 w-4 align-middle"
                />
              </TableCell>
              <TableCell>{item.name}</TableCell>
              <TableCell>
                {item.amount === 0
                  ? t("pages.shopping.amount_unspecified")
                  : formatQuantity(item.amount, item.unit)}
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
