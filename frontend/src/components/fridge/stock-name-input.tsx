import * as React from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { NameSuggestion } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { isImeComposing } from "./stock-form";

const MAX_SUGGESTIONS = 8;

type Props = {
  value: string;
  onChange: (value: string) => void;
  onSelect: (suggestion: NameSuggestion) => void;
  suggestions: NameSuggestion[];
  placeholder?: string;
  "aria-label"?: string;
  "aria-invalid"?: boolean;
  className?: string;
  autoFocus?: boolean;
};

// WAI-ARIA combobox パターンの候補補完つき自由入力 (design.md D6)。候補の選択は任意で、
// 候補にない名前もそのまま入力できる。
export const StockNameInput = React.forwardRef<HTMLInputElement, Props>(
  ({ value, onChange, onSelect, suggestions, className, ...inputProps }, ref) => {
    const { t } = useTranslation();
    const listboxId = React.useId();
    const [open, setOpen] = React.useState(false);
    const [activeIndex, setActiveIndex] = React.useState(-1);

    const matches = React.useMemo(() => {
      const q = value.trim().toLowerCase();
      if (q === "") return [];
      return suggestions.filter((s) => s.name.toLowerCase().includes(q)).slice(0, MAX_SUGGESTIONS);
    }, [value, suggestions]);

    const expanded = open && matches.length > 0;

    const select = (s: NameSuggestion) => {
      onSelect(s);
      setOpen(false);
      setActiveIndex(-1);
    };

    const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (isImeComposing(e)) return;
      switch (e.key) {
        case "ArrowDown":
          if (matches.length === 0) return;
          e.preventDefault();
          setOpen(true);
          setActiveIndex((i) => (i + 1) % matches.length);
          return;
        case "ArrowUp":
          if (!expanded) return;
          e.preventDefault();
          setActiveIndex((i) => (i <= 0 ? matches.length - 1 : i - 1));
          return;
        case "Enter":
          if (!expanded || activeIndex < 0) return;
          // 親の Enter 送信 / 保存より候補確定を優先させる。
          e.preventDefault();
          e.stopPropagation();
          select(matches[activeIndex]);
          return;
        case "Escape":
          if (!expanded) return;
          // 開いた候補を閉じるだけにとどめ、編集行のキャンセルまで伝播させない。
          e.preventDefault();
          e.stopPropagation();
          setOpen(false);
          setActiveIndex(-1);
          return;
      }
    };

    return (
      <div className={cn("relative", className)}>
        <Input
          ref={ref}
          {...inputProps}
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={listboxId}
          aria-activedescendant={
            expanded && activeIndex >= 0 ? `${listboxId}-${activeIndex}` : undefined
          }
          autoComplete="off"
          value={value}
          onChange={(e) => {
            onChange(e.target.value);
            setOpen(true);
            setActiveIndex(-1);
          }}
          onKeyDown={onKeyDown}
          onBlur={() => setOpen(false)}
        />
        {expanded && (
          <div
            id={listboxId}
            role="listbox"
            aria-label={t("pages.fridge.add.suggestions_label")}
            className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-md border border-neutral-200 bg-white py-1 text-sm shadow-md"
          >
            {matches.map((s, i) => (
              <div
                key={s.name}
                id={`${listboxId}-${i}`}
                role="option"
                // フォーカスは入力欄に残し、aria-activedescendant で選択中を伝える。
                tabIndex={-1}
                aria-selected={i === activeIndex}
                className={cn(
                  "flex cursor-pointer items-center justify-between gap-2 px-3 py-1.5",
                  i === activeIndex ? "bg-blue-50 text-blue-900" : "hover:bg-neutral-100",
                )}
                // blur より先に選択を確定させるため mousedown で拾う。
                onMouseDown={(e) => {
                  e.preventDefault();
                  select(s);
                }}
              >
                <span>{s.name}</span>
                {s.unit && <span className="text-xs text-neutral-500">{s.unit}</span>}
              </div>
            ))}
          </div>
        )}
      </div>
    );
  },
);
StockNameInput.displayName = "StockNameInput";
