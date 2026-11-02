import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Props = {
  tags: readonly string[];
  selected: readonly string[];
  onChange: (next: string[]) => void;
};

export function RecipeTagFilter({ tags, selected, onChange }: Props) {
  const { t } = useTranslation();
  if (tags.length === 0) return null;

  const toggle = (tag: string) => {
    onChange(selected.includes(tag) ? selected.filter((s) => s !== tag) : [...selected, tag]);
  };

  return (
    <fieldset
      aria-label={t("pages.recipes.tag_filter.aria_label")}
      className="m-0 flex min-w-0 flex-wrap items-center gap-2 border-0 p-0"
    >
      {tags.map((tag) => {
        const pressed = selected.includes(tag);
        return (
          <button
            key={tag}
            type="button"
            aria-pressed={pressed}
            onClick={() => toggle(tag)}
            className={cn(
              "rounded-full border px-3 py-0.5 text-sm transition-colors",
              pressed
                ? "border-blue-600 bg-blue-600 text-white"
                : "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-100",
            )}
          >
            {tag}
          </button>
        );
      })}
      {selected.length > 0 && (
        <Button type="button" variant="ghost" size="sm" onClick={() => onChange([])}>
          {t("pages.recipes.tag_filter.clear")}
        </Button>
      )}
    </fieldset>
  );
}
