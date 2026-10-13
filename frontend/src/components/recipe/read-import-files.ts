import type { ImportFile } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

// Go 側 recipeImportMaxFiles と揃える。Go 側でも件数を再確認する (add-recipe-bulk-import D2)。
export const RECIPE_IMPORT_MAX_FILES = 100;

export type ReadImportFilesResult =
  | { kind: "ok"; files: ImportFile[] }
  | { kind: "empty" }
  | { kind: "too_many" };

export async function readImportFiles(list: FileList | null): Promise<ReadImportFilesResult> {
  const markdown = Array.from(list ?? []).filter((f) => f.name.toLowerCase().endsWith(".md"));
  if (markdown.length === 0) return { kind: "empty" };
  if (markdown.length > RECIPE_IMPORT_MAX_FILES) return { kind: "too_many" };
  const files = await Promise.all(
    markdown.map(async (f) => ({
      name: f.webkitRelativePath !== "" ? f.webkitRelativePath : f.name,
      content: await f.text(),
    })),
  );
  return { kind: "ok", files };
}
