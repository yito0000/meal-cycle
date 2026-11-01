import { useTranslation } from "react-i18next";

export type SuggestionItem = { recipeId: string; recipeName: string; reason: string };

type Props = {
  title: string;
  countLabel: string;
  items: SuggestionItem[];
  onSelect: (item: SuggestionItem) => void;
};

// 軸に依らない候補の欄 1 つ分。軸を足すときはこの部品をもう 1 つ並べる。
export function SuggestionSection({ title, countLabel, items, onSelect }: Props) {
  const { t } = useTranslation();
  return (
    <section aria-label={title} data-testid="suggestion-section" className="flex flex-col gap-1">
      <h3 className="flex items-baseline justify-between text-xs font-semibold text-neutral-700">
        <span>{title}</span>
        <span className="font-normal text-neutral-500">{countLabel}</span>
      </h3>
      {items.length === 0 ? (
        <p className="px-1 text-sm text-neutral-500">
          {t("pages.meal_plan.picker.suggestion.none")}
        </p>
      ) : (
        <ul className="flex flex-col rounded-md border border-blue-100 bg-blue-50/40">
          {items.map((item) => (
            <li key={item.recipeId}>
              <button
                type="button"
                data-testid="suggestion-item"
                className="block w-full cursor-pointer px-3 py-1.5 text-left hover:bg-blue-50"
                onClick={() => onSelect(item)}
              >
                <span className="block text-sm">{item.recipeName}</span>
                <span className="block text-xs text-neutral-500">{item.reason}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
