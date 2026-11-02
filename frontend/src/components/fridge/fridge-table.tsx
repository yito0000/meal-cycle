import * as React from "react";
import { useTranslation } from "react-i18next";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type {
  NameSuggestion,
  StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { FridgeEditRow } from "./fridge-edit-row";
import { FridgeItemRow } from "./fridge-item-row";
import { FridgeReviewRow } from "./fridge-review-row";

export type StockReview = {
  reviewed: ReadonlySet<string>;
  onOk: (item: StockItem) => void;
  onReduced: (item: StockItem) => void;
  onGone: (item: StockItem) => void;
};

type Props = {
  items: StockItem[];
  nameSuggestions: NameSuggestion[];
  deleting: boolean;
  onDelete: (item: StockItem) => void;
  // 指定があるときは見直しの状態で描き、行内編集の「編集」「削除」は出さない (add-fridge-stock-review design.md D5)。
  review?: StockReview;
};

export function FridgeTable({ items, nameSuggestions, deleting, onDelete, review }: Props) {
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
          review ? (
            <FridgeReviewRow
              key={item.id}
              item={item}
              reviewed={review.reviewed.has(item.id)}
              disabled={deleting}
              onOk={() => review.onOk(item)}
              onReduced={() => review.onReduced(item)}
              onGone={() => review.onGone(item)}
            />
          ) : item.id === editingId ? (
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
