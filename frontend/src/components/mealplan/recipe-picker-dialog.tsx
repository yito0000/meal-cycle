import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { Input } from "../ui/input";

type PickedRecipe = { id: string; name: string };

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (picked: PickedRecipe) => void;
}

// レシピ選択ダイアログ (S02 のミニ版)。名前部分一致で絞り込み、行クリックで確定する。
// S02 の本画面には遷移せず、閉じたら呼び出し元のカレンダーに戻る。
export function RecipePickerDialog({ open, onOpenChange, onSelect }: Props) {
  const { t } = useTranslation();
  const [q, setQ] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["recipes", "list"],
    queryFn: () => RecipeService.List(),
    enabled: open,
  });

  const filtered = useMemo(() => {
    const list = data ?? [];
    if (q.trim() === "") return list;
    const needle = q.trim().toLowerCase();
    return list.filter((r) => r.name.toLowerCase().includes(needle));
  }, [data, q]);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) setQ("");
      }}
    >
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("pages.meal_plan.picker.title")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <Input
            placeholder={t("pages.meal_plan.picker.search_placeholder")}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            autoFocus
          />
          <ul className="max-h-64 overflow-y-auto rounded-md border border-neutral-200">
            {isLoading && (
              <li className="p-3 text-sm text-neutral-500">{t("pages.meal_plan.loading")}</li>
            )}
            {!isLoading && filtered.length === 0 && (
              <li className="p-3 text-sm text-neutral-500">{t("pages.meal_plan.picker.empty")}</li>
            )}
            {filtered.map((r) => (
              <li key={r.id}>
                <button
                  type="button"
                  className="block w-full cursor-pointer px-3 py-2 text-left text-sm hover:bg-neutral-100"
                  onClick={() => {
                    onSelect({ id: r.id, name: r.name });
                    onOpenChange(false);
                    setQ("");
                  }}
                >
                  {r.name}
                </button>
              </li>
            ))}
          </ul>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("pages.meal_plan.picker.cancel")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
