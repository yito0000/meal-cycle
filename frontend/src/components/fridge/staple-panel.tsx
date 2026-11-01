import * as React from "react";
import { useTranslation } from "react-i18next";
import { IngredientNameInput } from "@/components/ingredient/ingredient-name-input";
import { Button } from "@/components/ui/button";
import { isImeComposing } from "@/lib/ime";
import type {
  NameSuggestion,
  Staple,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { stapleErrorOf } from "./error-messages";
import { FieldError } from "./field-errors";
import { StapleRow } from "./staple-row";
import { useAddStaple, useStapleList } from "./use-staples";

type Props = {
  nameSuggestions: NameSuggestion[];
  deleting: boolean;
  onDelete: (item: Staple) => void;
};

// S05 の常備品の欄。検索と並び替えは効かせず、食材名の順で全件を並べる (add-fridge-staples design.md D7)。
export function StaplePanel({ nameSuggestions, deleting, onDelete }: Props) {
  const { t } = useTranslation();
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [name, setName] = React.useState("");
  const [error, setError] = React.useState<string>();
  const { data, isError } = useStapleList();
  const add = useAddStaple();

  const sorted = React.useMemo(
    () => [...(data ?? [])].sort((a, b) => a.name.localeCompare(b.name, "ja")),
    [data],
  );

  const submit = () => {
    if (add.isPending) return;
    if (name.trim() === "") {
      setError("pages.fridge.staples.errors.name_empty");
      return;
    }
    setError(undefined);
    add.mutate(name, {
      onSuccess: () => {
        setName("");
        inputRef.current?.focus();
      },
      onError: (err) => setError(stapleErrorOf(err)),
    });
  };

  // 候補の確定に使った Enter は IngredientNameInput が preventDefault するので、ここには届かない。
  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Enter" || e.defaultPrevented || isImeComposing(e)) return;
    e.preventDefault();
    submit();
  };

  return (
    <section
      aria-labelledby="staple-panel-title"
      className="flex flex-col gap-3 rounded-md border border-neutral-200 bg-white p-3"
    >
      <div>
        <h2 id="staple-panel-title" className="text-base font-semibold">
          {t("pages.fridge.staples.title")}
        </h2>
        <p className="text-xs text-neutral-500">{t("pages.fridge.staples.description")}</p>
      </div>

      <div>
        {/* biome-ignore lint/a11y/noStaticElementInteractions: 入力欄の Enter をまとめて受ける */}
        <div className="flex items-start gap-2" onKeyDown={onKeyDown}>
          <IngredientNameInput
            ref={inputRef}
            className="min-w-0 flex-1"
            value={name}
            onChange={(v) => {
              setName(v);
              setError(undefined);
            }}
            onSelect={(s) => {
              setName(s.name);
              setError(undefined);
            }}
            suggestions={nameSuggestions}
            placeholder={t("pages.fridge.staples.name_placeholder")}
            aria-label={t("pages.fridge.staples.name_placeholder")}
            aria-invalid={!!error}
          />
          <Button type="button" onClick={submit} disabled={add.isPending}>
            {t("pages.fridge.staples.add")}
          </Button>
        </div>
        <FieldError messageKey={error} />
      </div>

      {isError && <p className="text-sm text-red-700">{t("pages.fridge.load_error")}</p>}
      {data && sorted.length === 0 && (
        <p className="text-sm text-neutral-500">{t("pages.fridge.staples.empty")}</p>
      )}
      {sorted.length > 0 && (
        <ul aria-label={t("pages.fridge.staples.title")}>
          {sorted.map((s) => (
            <StapleRow key={s.id} item={s} deleting={deleting} onDelete={onDelete} />
          ))}
        </ul>
      )}
    </section>
  );
}
