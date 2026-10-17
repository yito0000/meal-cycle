import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FieldError } from "@/components/fridge/field-errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { isImeComposing } from "@/lib/ime";
import { Chip } from "./chip";

type Props = {
  label: string;
  hint: string;
  values: readonly number[];
  onChange: (next: number[]) => void;
  // 消す操作を出さない値。
  lockedValues?: readonly number[];
  // 保存時に Go が返したこの欄のエラー (i18n key)。
  errorKey?: string;
};

// 追加時に確かめるのは整数かと重複だけ。件数と範囲は保存時に Go が判定する (design.md D5)。
export function NumberListField({
  label,
  hint,
  values,
  onChange,
  lockedValues = [],
  errorKey,
}: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");
  const [addError, setAddError] = useState<string | undefined>();

  const add = () => {
    const text = draft.trim();
    if (!/^\d+$/.test(text)) {
      setAddError("pages.settings.errors.not_integer");
      return;
    }
    const n = Number(text);
    if (values.includes(n)) {
      setAddError("pages.settings.errors.day_duplicate");
      return;
    }
    setAddError(undefined);
    setDraft("");
    onChange([...values, n].sort((a, b) => a - b));
  };

  return (
    <fieldset className="flex min-w-0 flex-col gap-2">
      <legend className="mb-2 text-sm font-medium text-neutral-800">{label}</legend>
      <ul className="flex flex-wrap items-center gap-2">
        {values.map((v) => (
          <Chip
            key={v}
            label={t("pages.settings.day_chip", { count: v })}
            onRemove={
              lockedValues.includes(v) ? undefined : () => onChange(values.filter((x) => x !== v))
            }
          />
        ))}
      </ul>
      <div className="flex items-center gap-2">
        <Input
          type="text"
          inputMode="numeric"
          value={draft}
          aria-label={`${label} ${t("pages.settings.add_placeholder.days")}`}
          placeholder={t("pages.settings.add_placeholder.days")}
          onChange={(e) => {
            setDraft(e.target.value);
            setAddError(undefined);
          }}
          onKeyDown={(e) => {
            if (e.key !== "Enter" || isImeComposing(e)) return;
            e.preventDefault();
            add();
          }}
          className="w-24"
        />
        <Button type="button" variant="outline" size="sm" onClick={add}>
          {t("pages.settings.add")}
        </Button>
      </div>
      <FieldError messageKey={addError ?? errorKey} />
      <p className="text-xs text-neutral-500">{hint}</p>
    </fieldset>
  );
}
