import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { TableCell, TableRow } from "@/components/ui/table";
import { formatQuantity } from "@/lib/format-quantity";
import { cn } from "@/lib/utils";
import {
  ExpiryStatus,
  type StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { formatDaysLeft } from "./days-left";
import { ExpiryDot } from "./expiry-dot";

type Props = {
  item: StockItem;
  disabled: boolean;
  onEdit: () => void;
  onDelete: () => void;
};

export function FridgeItemRow({ item, disabled, onEdit, onDelete }: Props) {
  const { t } = useTranslation();
  const none = t("pages.fridge.days_left.none");
  const expired = item.expiryStatus === ExpiryStatus.ExpiryExpired;
  return (
    <TableRow
      data-stock-id={item.id}
      data-expired={expired || undefined}
      className={cn(expired && "bg-red-50 hover:bg-red-100/70")}
    >
      <TableCell className="font-medium">
        <span className="flex items-center gap-2">
          <ExpiryDot status={item.expiryStatus} />
          <span>{item.name}</span>
        </span>
      </TableCell>
      <TableCell className="tabular-nums">{formatQuantity(item.amount, item.unit)}</TableCell>
      <TableCell className="tabular-nums text-neutral-700">{item.expiresAt || none}</TableCell>
      <TableCell
        className={cn(
          "tabular-nums",
          expired || item.expiryStatus === ExpiryStatus.ExpiryUrgent
            ? "text-red-700"
            : "text-neutral-700",
          expired && "font-semibold",
        )}
      >
        {formatDaysLeft(t, item.daysLeft)}
      </TableCell>
      <TableCell>
        <div className="flex gap-2">
          <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={onEdit}>
            {t("pages.fridge.row.edit")}
          </Button>
          <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={onDelete}>
            {t("pages.fridge.row.delete")}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
