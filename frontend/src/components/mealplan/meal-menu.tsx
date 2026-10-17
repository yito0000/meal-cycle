import { useTranslation } from "react-i18next";
import { useSettingsValues } from "@/components/settings/settings-gate";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

interface Props {
  onClear: () => void;
  onCopyDays: (days: number) => void;
}

// 時間帯右下の `⋯` メニュー: 全クリア / 全体連続 N 日コピー。
export function MealMenu({ onClear, onCopyDays }: Props) {
  const { t } = useTranslation();
  const { leftoverCopyPresets } = useSettingsValues();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("pages.meal_plan.cell_actions.meal_menu")}
        className="px-1 text-neutral-400 hover:text-neutral-700"
      >
        ⋯
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onClear}>
          {t("pages.meal_plan.meal_menu.clear")}
        </DropdownMenuItem>
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            {t("pages.meal_plan.meal_menu.copy_days", { days: "N" })}
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent>
            {leftoverCopyPresets.map((n) => (
              <DropdownMenuItem key={n} onClick={() => onCopyDays(n)}>
                {t("pages.meal_plan.copy.days_option", { days: n })}
              </DropdownMenuItem>
            ))}
          </DropdownMenuSubContent>
        </DropdownMenuSub>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
