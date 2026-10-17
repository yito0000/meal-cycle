import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FieldError } from "@/components/fridge/field-errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { isImeComposing } from "@/lib/ime";
import { Chip } from "./chip";

// internal/domain/settings/validate.go の WordRunesMax と同じ値。追加の時点で気付けるよう写している (design.md D5)。
const WORD_RUNES_MAX = 20;

type Props = {
  label: string;
  hint: string;
  placeholder: string;
  values: readonly string[];
  onChange: (next: string[]) => void;
  lengthErrorKey: string;
  duplicateErrorKey: string;
  errorKey?: string;
};

// 追加時に確かめるのは空白、文字数、重複だけ。件数は保存時に Go が判定する (design.md D5)。
export function WordListField({
  label,
  hint,
  placeholder,
  values,
  onChange,
  lengthErrorKey,
  duplicateErrorKey,
  errorKey,
}: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");
  const [addError, setAddError] = useState<string | undefined>();

  const add = () => {
    const word = draft.trim();
    if (word === "") {
      setAddError("pages.settings.errors.blank");
      return;
    }
    // 絵文字などのサロゲートペアを 1 文字と数える Go の rune 数にそろえる。
    if ([...word].length > WORD_RUNES_MAX) {
      setAddError(lengthErrorKey);
      return;
    }
    if (values.includes(word)) {
      setAddError(duplicateErrorKey);
      return;
    }
    setAddError(undefined);
    setDraft("");
    onChange([...values, word]);
  };

  return (
    <fieldset className="flex min-w-0 flex-col gap-2">
      <legend className="mb-2 text-sm font-medium text-neutral-800">{label}</legend>
      <ul className="flex flex-wrap items-center gap-2">
        {values.map((v) => (
          <Chip key={v} label={v} onRemove={() => onChange(values.filter((x) => x !== v))} />
        ))}
      </ul>
      <div className="flex items-center gap-2">
        <Input
          type="text"
          value={draft}
          aria-label={`${label} ${placeholder}`}
          placeholder={placeholder}
          onChange={(e) => {
            setDraft(e.target.value);
            setAddError(undefined);
          }}
          onKeyDown={(e) => {
            if (e.key !== "Enter" || isImeComposing(e)) return;
            e.preventDefault();
            add();
          }}
          className="w-48"
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
