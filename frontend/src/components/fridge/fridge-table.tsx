import * as React from "react";
import { useTranslation } from "react-i18next";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type {
  NameSuggestion,
  StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { FridgeEditRow } from "./fridge-edit-row";
import { FridgeItemRow } from "./fridge-item-row";

type Props = {
  items: StockItem[];
  nameSuggestions: NameSuggestion[];
  deleting: boolean;
  onDelete: (item: StockItem) => void;
};

export function FridgeTable({ items, nameSuggestions, deleting, onDelete }: Props) {
  const { t } = useTranslation();
  // 行内編集は同時に 1 行まで。編集中は他行の編集 / 削除を無効にする (design.md D10)。
  const [editingId, setEditingId] = React.useState<string | null>(null);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[32%]">{t("pages.fridge.column.name")}</TableHead>
          <TableHead className="w-[18%]">{t("pages.fridge.column.amount")}</TableHead>
          <TableHead className="w-[18%]">{t("pages.fridge.column.expires_at")}</TableHead>
          <TableHead className="w-[12%]">{t("pages.fridge.column.days_left")}</TableHead>
          <TableHead className="w-[20%]">{t("pages.fridge.column.actions")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) =>
          item.id === editingId ? (
            <FridgeEditRow
              key={item.id}
              item={item}
              nameSuggestions={nameSuggestions}
              onDone={() => setEditingId(null)}
            />
          ) : (
            <FridgeItemRow
              key={item.id}
              item={item}
              disabled={editingId !== null || deleting}
              onEdit={() => setEditingId(item.id)}
              onDelete={() => onDelete(item)}
            />
          ),
        )}
      </TableBody>
    </Table>
  );
}
