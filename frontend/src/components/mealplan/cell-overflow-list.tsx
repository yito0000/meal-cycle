import { useTranslation } from "react-i18next";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { MealItemRow } from "./meal-item-row";
import type { EntryAction } from "./meal-plan-cell";

interface Props {
  sortedEntries: Entry[];
  overflow: number;
  label: string;
  onEntryAction: (action: EntryAction, entry: Entry) => void;
}

// 畳まれた品だけでなく全品を出すのは、「上へ移動」で品が上限の内側へ入っても一覧から消えないようにするため
// (add-mealplan-cell-overflow-list design.md D2)。浮いた面なので一覧の中ではドラッグさせない。
export function CellOverflowList({ sortedEntries, overflow, label, onEntryAction }: Props) {
  const { t } = useTranslation();
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="self-start rounded px-0.5 text-left text-xs text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800"
        >
          {t("pages.meal_plan.cell.more_items", { count: overflow })}
        </button>
      </PopoverTrigger>
      <PopoverContent
        side="right"
        align="start"
        className="w-56 p-1"
        aria-label={label}
        data-testid="cell-overflow-list"
      >
        <div className="flex flex-col gap-1">
          {sortedEntries.map((entry) => (
            <MealItemRow
              key={entry.id}
              entry={entry}
              cellItemCount={sortedEntries.length}
              fontClass="text-sm"
              draggable={false}
              isDragSource={false}
              onAction={onEntryAction}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}
