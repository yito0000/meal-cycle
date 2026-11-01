import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import type { Staple } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stapleErrorOf } from "./error-messages";
import { StapleStateToggle } from "./staple-state-toggle";
import { useSetStapleState } from "./use-staples";

type Props = {
  item: Staple;
  deleting: boolean;
  onDelete: (item: Staple) => void;
};

export function StapleRow({ item, deleting, onDelete }: Props) {
  const { t } = useTranslation();
  const setState = useSetStapleState();
  return (
    <li
      data-testid="staple-row"
      data-state={item.state}
      className="flex items-center gap-2 border-b border-neutral-100 py-1.5 last:border-b-0"
    >
      <span className="min-w-0 flex-1 truncate text-sm">{item.name}</span>
      <StapleStateToggle
        value={item.state}
        name={item.name}
        // 保存中に押し直すと順序が入れ替わるので、その行の切替を止める。
        disabled={setState.isPending}
        onChange={(state) =>
          setState.mutate(
            { id: item.id, state },
            { onError: (err) => toast.error(t(stapleErrorOf(err))) },
          )
        }
      />
      <Button
        type="button"
        variant="ghost"
        size="sm"
        disabled={deleting}
        onClick={() => onDelete(item)}
      >
        {t("pages.fridge.staples.delete")}
      </Button>
    </li>
  );
}
