import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
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
import type {
  ImportCandidate,
  ImportFile,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { RecipeImportList } from "./recipe-import-list";
import { RecipeImportPreview } from "./recipe-import-preview";

type Props = {
  // 選び直すたびに増やし、同じ中身のファイルでも読み直させる。
  requestId: number;
  files: ImportFile[];
  onClose: () => void;
};

export function RecipeImportDialog({ requestId, files, onClose }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const preview = useQuery({
    queryKey: ["recipes", "import-preview", requestId],
    queryFn: async () => (await RecipeService.PreviewImport(files)) ?? [],
    gcTime: 0,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
  const candidates = preview.data ?? [];

  // 既定は問題も同名も無い候補。ユーザーが触るまでは既定を使う (add-recipe-bulk-import D8)。
  const [touched, setTouched] = useState<Set<string> | null>(null);
  const defaultSelected = useMemo(
    () => new Set(candidates.filter((c) => !c.issue && !c.duplicate).map((c) => c.fileName)),
    [candidates],
  );
  const selected = touched ?? defaultSelected;

  const single = candidates.length === 1 ? candidates[0] : undefined;
  const toImport: ImportCandidate[] = single
    ? single.issue
      ? []
      : [single]
    : candidates.filter((c) => !c.issue && selected.has(c.fileName));

  const importMutation = useMutation({
    mutationFn: () => RecipeService.Import(toImport.map((c) => c.recipe)),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ["recipes", "list"] });
      toast(t("pages.recipes.import.done", { count: created?.length ?? 0 }));
      onClose();
    },
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[85vh] max-w-4xl flex-col">
        <DialogHeader>
          <DialogTitle>{t("pages.recipes.import.title")}</DialogTitle>
          <DialogDescription className="sr-only">
            {t("pages.recipes.import.title")}
          </DialogDescription>
        </DialogHeader>

        {preview.isLoading && (
          <p className="text-sm text-neutral-500">{t("pages.recipes.import.loading")}</p>
        )}
        {preview.isError && (
          <p className="text-sm text-red-700">{t("pages.recipes.import.load_error")}</p>
        )}
        {single && <RecipeImportPreview candidate={single} />}
        {candidates.length > 1 && (
          <RecipeImportList
            candidates={candidates}
            selected={selected}
            onSelectedChange={setTouched}
          />
        )}

        {importMutation.isError && (
          <p className="text-sm text-red-700">{t("pages.recipes.import.error")}</p>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("pages.recipes.import.cancel")}
          </Button>
          <Button
            type="button"
            disabled={toImport.length === 0 || importMutation.isPending}
            onClick={() => importMutation.mutate()}
          >
            {single
              ? t("pages.recipes.import.submit")
              : t("pages.recipes.import.submit_count", { count: toImport.length })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
