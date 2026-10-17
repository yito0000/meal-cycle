import { X } from "lucide-react";
import { useTranslation } from "react-i18next";

type Props = {
  label: string;
  // 未指定なら消す操作を出さない (買い物リストの 7 / 14 / 28 日)。
  onRemove?: () => void;
};

export function Chip({ label, onRemove }: Props) {
  const { t } = useTranslation();
  return (
    <li
      data-chip={label}
      className="inline-flex items-center gap-1 rounded-full border border-neutral-300 bg-white py-0.5 pr-1 pl-3 text-sm"
    >
      <span className={onRemove ? undefined : "pr-2"}>{label}</span>
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={t("pages.settings.remove", { value: label })}
          className="rounded-full p-0.5 text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </li>
  );
}
