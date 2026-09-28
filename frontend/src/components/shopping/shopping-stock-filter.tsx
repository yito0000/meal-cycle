import { useTranslation } from "react-i18next";
import { STOCK_FILTERS, type StockFilter } from "./stock-filter";

type Props = {
  value: StockFilter;
  counts: Record<StockFilter, number>;
  onChange: (next: StockFilter) => void;
};

export function ShoppingStockFilter({ value, counts, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <fieldset className="inline-flex w-fit items-center rounded-md border border-neutral-200 bg-white text-sm">
      <legend className="sr-only">{t("pages.shopping.filter.label")}</legend>
      {STOCK_FILTERS.map((filter) => {
        const isActive = filter === value;
        return (
          <button
            key={filter}
            type="button"
            onClick={() => onChange(filter)}
            aria-pressed={isActive}
            data-testid={`shopping-filter-${filter}`}
            className={`px-3 py-1.5 first:rounded-l-md last:rounded-r-md transition-colors ${
              isActive ? "bg-blue-600 text-white" : "text-neutral-800 hover:bg-neutral-100"
            }`}
          >
            {t(`pages.shopping.filter.${filter}`, { count: counts[filter] })}
          </button>
        );
      })}
    </fieldset>
  );
}
