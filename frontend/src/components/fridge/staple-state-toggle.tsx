import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { StapleState } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";

type Props = {
  value: StapleState;
  disabled: boolean;
  onChange: (state: StapleState) => void;
  name: string;
};

const OPTIONS: { state: StapleState; key: string; active: string }[] = [
  { state: StapleState.StapleEnough, key: "enough", active: "bg-neutral-700 text-white" },
  { state: StapleState.StapleLow, key: "low", active: "bg-amber-400 text-amber-950" },
  { state: StapleState.StapleOut, key: "out", active: "bg-red-600 text-white" },
];

// 1 回押せば目的の状態になる 3 択 (add-fridge-staples design.md D8)。色だけに頼らず文字で状態を示す。
export function StapleStateToggle({ value, disabled, onChange, name }: Props) {
  const { t } = useTranslation();
  return (
    <fieldset
      aria-label={t("pages.fridge.staples.state_group", { name })}
      className="m-0 inline-flex min-w-0 overflow-hidden rounded-md border border-neutral-300 p-0 text-xs"
    >
      {OPTIONS.map((o) => {
        const pressed = o.state === value;
        return (
          <button
            key={o.state}
            type="button"
            aria-pressed={pressed}
            disabled={disabled}
            onClick={() => {
              if (!pressed) onChange(o.state);
            }}
            className={cn(
              "px-2 py-1 font-medium transition-colors disabled:opacity-60",
              pressed ? o.active : "bg-white text-neutral-600 hover:bg-neutral-100",
            )}
          >
            {t(`pages.fridge.staples.state.${o.key}`)}
          </button>
        );
      })}
    </fieldset>
  );
}
