import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  canSave: boolean;
  onSave: () => void;
  onOpenList: () => void;
};

// 下部の CTA が増えすぎないよう、保存と適用を 1 つのメニューにまとめる (add-mealplan-template design.md D3)。
export function TemplateMenu({ canSave, onSave, onOpenList }: Props) {
  const { t } = useTranslation();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" data-testid="template-menu">
          {t("pages.meal_plan.template.menu")}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem disabled={!canSave} onSelect={onSave}>
          {t("pages.meal_plan.template.save_week")}
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={onOpenList}>
          {t("pages.meal_plan.template.open_list")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
