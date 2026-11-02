import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { SpreadResult } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { errorI18nKey } from "./error-messages";
import { mealLabelKey } from "./mealplan-config";
import { type TemplateView, useDeleteTemplate, useTemplates } from "./use-templates";

type Props = {
  open: boolean;
  weekStart: string;
  onOpenChange: (open: boolean) => void;
  onApplied: (result: SpreadResult, count: number, skipped: number) => void;
};

// 曜日の名前は表示言語の Intl に任せる。2024-01-01 は月曜なので、ずれ 0〜6 を月〜日に当てる。
function weekdayName(offset: number, language: string): string {
  return new Intl.DateTimeFormat(language, { weekday: "short" }).format(
    new Date(2024, 0, 1 + offset),
  );
}

// 曜日 × 時間帯ごとに品の名前をまとめる。items は (日, 時間帯, 順) に並んでいる。
function cellsOf(tpl: TemplateView) {
  const cells: {
    key: string;
    offset: number;
    meal: string;
    names: { name: string; archived: boolean; position: number }[];
  }[] = [];
  for (const it of tpl.items) {
    const key = `${it.dayOffset}|${it.meal}`;
    let cell = cells.find((c) => c.key === key);
    if (!cell) {
      cell = { key, offset: it.dayOffset, meal: it.meal, names: [] };
      cells.push(cell);
    }
    cell.names.push({ name: it.recipeName, archived: it.archived, position: it.position });
  }
  return cells;
}

export function TemplateListDialog({ open, weekStart, onOpenChange, onApplied }: Props) {
  const { t, i18n } = useTranslation();
  const { data, isLoading, isError } = useTemplates(open);
  const del = useDeleteTemplate();
  const [pending, setPending] = useState(false);
  const [confirming, setConfirming] = useState<TemplateView | null>(null);

  const apply = async (tpl: TemplateView) => {
    if (pending) return;
    setPending(true);
    try {
      const res = await MealPlanService.ApplyTemplate(tpl.id, weekStart);
      onApplied(res.result, res.result.addedEntryIds?.length ?? 0, res.skippedArchived);
    } catch (err) {
      toast.error(t("pages.meal_plan.template.apply_failed"), {
        description: t(errorI18nKey(err)),
      });
    } finally {
      setPending(false);
    }
  };

  const confirmDelete = () => {
    if (!confirming) return;
    del.mutate(confirming.id, {
      onSuccess: () => setConfirming(null),
      onError: (err) => toast.error(t(errorI18nKey(err))),
    });
  };

  const templates = data ?? [];
  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[85vh] max-w-2xl flex-col">
          <DialogHeader>
            <DialogTitle>{t("pages.meal_plan.template.list_title")}</DialogTitle>
            <DialogDescription>{t("pages.meal_plan.template.list_description")}</DialogDescription>
          </DialogHeader>
          <div className="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
            {isLoading && (
              <p className="text-sm text-neutral-500">{t("pages.meal_plan.loading")}</p>
            )}
            {isError && (
              <p className="text-sm text-red-700">{t("pages.meal_plan.template.load_error")}</p>
            )}
            {data && templates.length === 0 && (
              <p className="text-sm text-neutral-500">{t("pages.meal_plan.template.empty")}</p>
            )}
            <div className="flex flex-col gap-3">
              {templates.map((tpl) => {
                const allArchived = tpl.items.every((it) => it.archived);
                return (
                  <section
                    key={tpl.id}
                    data-testid="template"
                    aria-label={tpl.name}
                    className="flex flex-col gap-2 rounded-md border border-neutral-200 p-3"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <h3 className="text-sm font-semibold">
                        {tpl.name}
                        <span className="ml-2 font-normal text-neutral-600">
                          {t("pages.meal_plan.template.item_count", { count: tpl.items.length })}
                        </span>
                      </h3>
                      <div className="flex gap-2">
                        <Button
                          type="button"
                          size="sm"
                          disabled={pending || allArchived}
                          onClick={() => apply(tpl)}
                        >
                          {t("pages.meal_plan.template.apply")}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          onClick={() => setConfirming(tpl)}
                        >
                          {t("pages.meal_plan.template.delete")}
                        </Button>
                      </div>
                    </div>
                    <ul className="flex flex-col gap-1">
                      {cellsOf(tpl).map((cell) => (
                        <li
                          key={cell.key}
                          data-testid="template-cell"
                          className="grid grid-cols-[6rem_1fr] gap-x-3 text-sm"
                        >
                          <span className="text-neutral-600">
                            {weekdayName(cell.offset, i18n.language)} {t(mealLabelKey(cell.meal))}
                          </span>
                          <span>
                            {cell.names.map((n, i) => (
                              <span
                                key={`${cell.key}-${n.position}`}
                                className={n.archived ? "text-neutral-400" : ""}
                              >
                                {i > 0 && t("pages.meal_plan.picker.suggestion.name_separator")}
                                {n.archived
                                  ? t("pages.meal_plan.template.archived_item", { name: n.name })
                                  : n.name}
                              </span>
                            ))}
                          </span>
                        </li>
                      ))}
                    </ul>
                  </section>
                );
              })}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog
        open={confirming !== null}
        onOpenChange={(next) => {
          if (!next && !del.isPending) setConfirming(null);
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {t("pages.meal_plan.template.delete_confirm.title", { name: confirming?.name ?? "" })}
            </DialogTitle>
            <DialogDescription>
              {t("pages.meal_plan.template.delete_confirm.body")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={del.isPending}
              onClick={() => setConfirming(null)}
            >
              {t("pages.meal_plan.template.delete_confirm.cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={del.isPending}
              onClick={confirmDelete}
            >
              {t("pages.meal_plan.template.delete_confirm.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
