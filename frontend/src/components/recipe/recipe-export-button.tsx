import { useMutation } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export function RecipeExportButton({ disabled }: { disabled: boolean }) {
  const { t } = useTranslation();
  const exportMutation = useMutation({
    mutationFn: () => RecipeService.ExportToFolder(t("pages.recipes.export.dialog_title")),
    onSuccess: (result) => {
      if (result.cancelled) return;
      toast(t("pages.recipes.export.done", { count: result.count }), { description: result.dir });
    },
    onError: (err) => {
      // Wails は型付きエラーを運ばないので、Go のエラー文字列の部分一致で見分ける。
      const msg = err instanceof Error ? err.message : String(err ?? "");
      if (msg.includes("recipe export empty")) {
        toast.error(t("pages.recipes.export.empty"));
        return;
      }
      // 権限が無いのかディスクが一杯なのかを見分けられるよう、OS の理由をそのまま添える (add-recipe-export D6)。
      toast.error(t("pages.recipes.export.failed"), { description: msg });
    },
  });

  return (
    <Button
      type="button"
      variant="outline"
      // ダイアログを開いている間も実行中に含め、二重に開かせない。
      disabled={disabled || exportMutation.isPending}
      onClick={() => exportMutation.mutate()}
    >
      {t("pages.recipes.export.button")}
    </Button>
  );
}
