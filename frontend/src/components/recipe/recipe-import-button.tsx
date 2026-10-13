import { type ChangeEvent, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import type { ImportFile } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { RECIPE_IMPORT_MAX_FILES, readImportFiles } from "./read-import-files";

// React の型定義は webkitdirectory を持たないので、属性として広げて渡す (add-recipe-bulk-import D8)。
const directoryAttrs = { webkitdirectory: "" } as Record<string, string>;

export function RecipeImportButton({ onRead }: { onRead: (files: ImportFile[]) => void }) {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const handleChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const input = e.currentTarget;
    const result = await readImportFiles(input.files);
    // 同じファイルを選び直しても change が発火するよう空にする。
    input.value = "";
    if (result.kind === "empty") {
      toast(t("pages.recipes.import.no_files"));
      return;
    }
    if (result.kind === "too_many") {
      toast(t("pages.recipes.import.too_many_files", { max: RECIPE_IMPORT_MAX_FILES }));
      return;
    }
    onRead(result.files);
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="outline">
            {t("pages.recipes.import.button")}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={() => fileInputRef.current?.click()}>
            {t("pages.recipes.import.choose_files")}
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => folderInputRef.current?.click()}>
            {t("pages.recipes.import.choose_folder")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept=".md"
        hidden
        data-testid="recipe-import-files"
        onChange={handleChange}
      />
      <input
        ref={folderInputRef}
        type="file"
        hidden
        data-testid="recipe-import-folder"
        onChange={handleChange}
        {...directoryAttrs}
      />
    </>
  );
}
