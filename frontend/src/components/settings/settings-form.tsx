import { type ReactNode, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { FieldError } from "@/components/fridge/field-errors";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { mealLabelKey } from "../mealplan/mealplan-config";
import { type SettingsError, type SettingsField, settingsErrorOf } from "./error-messages";
import { NumberListField } from "./number-list-field";
import {
  type SettingsConstraints,
  type SettingsValues,
  toSettingsValues,
  useResetSettings,
  useSaveSettings,
} from "./use-settings";
import { WordListField } from "./word-list-field";

// 今の表示言語にかかわらず各言語の名前で出すので、i18n を通さない (spec「S09 設定画面の表示」)。
const LANGUAGE_OPTIONS = [
  { value: "ja", label: "日本語" },
  { value: "en", label: "English" },
] as const;

// 数値の欄は入力途中の空欄や小数を持てるよう文字列で持ち、保存時に整数へ直す。
type Draft = {
  language: string;
  expiryRed: string;
  expiryYellow: string;
  leftoverPresets: number[];
  shoppingPresets: number[];
  shoppingDefault: number;
  measuringUnits: string[];
  unquantifiedWords: string[];
  importServings: string;
  recencyDays: string;
  weeklyFillDays: number[];
  weeklyFillMeal: string;
  weeklyRunDays: string;
};

function toDraft(v: SettingsValues): Draft {
  return {
    language: v.language,
    expiryRed: String(v.expiryAlertRedDays),
    expiryYellow: String(v.expiryAlertYellowDays),
    leftoverPresets: [...v.leftoverCopyPresets],
    shoppingPresets: [...v.shoppingPeriodDayPresets],
    shoppingDefault: v.shoppingDefaultPeriodDays,
    measuringUnits: [...v.measuringUnits],
    unquantifiedWords: [...v.unquantifiedWords],
    importServings: String(v.recipeImportDefaultServings),
    recencyDays: String(v.recencyThresholdDays),
    weeklyFillDays: [...v.weeklyFillDays],
    weeklyFillMeal: v.weeklyFillMeal,
    weeklyRunDays: String(v.weeklyPresetMaxRunDays),
  };
}

function parseInteger(s: string): number | null {
  const text = s.trim();
  return /^-?\d+$/.test(text) ? Number(text) : null;
}

type ParseResult = { ok: true; values: SettingsValues } | { ok: false; error: SettingsError };

// 整数に読めない欄は JSON の int に載らないので送らない。範囲や項目間の条件は Go が判定する (design.md D5)。
function fromDraft(d: Draft): ParseResult {
  const numbers: [SettingsField, string][] = [
    ["expiryRed", d.expiryRed],
    ["expiryYellow", d.expiryYellow],
    ["importServings", d.importServings],
    ["recencyDays", d.recencyDays],
    ["weeklyRunDays", d.weeklyRunDays],
  ];
  for (const [field, text] of numbers) {
    if (parseInteger(text) === null) {
      return { ok: false, error: { field, key: "pages.settings.errors.not_integer" } };
    }
  }
  return {
    ok: true,
    values: {
      language: d.language,
      expiryAlertRedDays: parseInteger(d.expiryRed) ?? 0,
      expiryAlertYellowDays: parseInteger(d.expiryYellow) ?? 0,
      leftoverCopyPresets: [...d.leftoverPresets].sort((a, b) => a - b),
      shoppingPeriodDayPresets: [...d.shoppingPresets].sort((a, b) => a - b),
      shoppingDefaultPeriodDays: d.shoppingDefault,
      measuringUnits: d.measuringUnits.map((w) => w.trim()),
      unquantifiedWords: d.unquantifiedWords.map((w) => w.trim()),
      recipeImportDefaultServings: parseInteger(d.importServings) ?? 0,
      recencyThresholdDays: parseInteger(d.recencyDays) ?? 0,
      weeklyFillDays: [...d.weeklyFillDays].sort((a, b) => a - b),
      weeklyFillMeal: d.weeklyFillMeal,
      weeklyPresetMaxRunDays: parseInteger(d.weeklyRunDays) ?? 0,
    },
  };
}

// 並べ替えや前後の空白だけの違いは変更とみなさない (design.md D10)。
function sameValues(a: SettingsValues, b: SettingsValues): boolean {
  const canon = (v: SettingsValues) =>
    JSON.stringify([
      v.language,
      v.expiryAlertRedDays,
      v.expiryAlertYellowDays,
      [...v.leftoverCopyPresets].sort((x, y) => x - y),
      [...v.shoppingPeriodDayPresets].sort((x, y) => x - y),
      v.shoppingDefaultPeriodDays,
      v.measuringUnits.map((w) => w.trim()),
      v.unquantifiedWords.map((w) => w.trim()),
      v.recipeImportDefaultServings,
      v.recencyThresholdDays,
      [...v.weeklyFillDays].sort((x, y) => x - y),
      v.weeklyFillMeal,
      v.weeklyPresetMaxRunDays,
    ]);
  return canon(a) === canon(b);
}

export function SettingsForm({
  saved,
  constraints,
}: {
  saved: SettingsValues;
  constraints: SettingsConstraints;
}) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState<Draft>(() => toDraft(saved));
  const [error, setError] = useState<SettingsError | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const save = useSaveSettings();
  const reset = useResetSettings();

  const parsed = fromDraft(draft);
  const dirty = !parsed.ok || !sameValues(parsed.values, saved);
  const pending = save.isPending || reset.isPending;

  const update = (patch: Partial<Draft>) => {
    setDraft((d) => ({ ...d, ...patch }));
    setError(null);
  };
  const errorFor = (field: SettingsField) => (error?.field === field ? error.key : undefined);

  const onSave = () => {
    if (!parsed.ok) {
      setError(parsed.error);
      return;
    }
    save.mutate(parsed.values, {
      onSuccess: (next) => {
        setDraft(toDraft(toSettingsValues(next)));
        setError(null);
        toast(t("pages.settings.toast.saved"));
      },
      onError: (err) => {
        const e = settingsErrorOf(err);
        if (e.field) setError(e);
        else toast.error(t(e.key));
      },
    });
  };

  const onReset = () => {
    reset.mutate(undefined, {
      onSuccess: (next) => {
        setConfirmOpen(false);
        setDraft(toDraft(toSettingsValues(next)));
        setError(null);
        toast(t("pages.settings.toast.reset"));
      },
      onError: () => toast.error(t("pages.settings.toast.reset_failed")),
    });
  };

  const onShoppingPresetsChange = (next: number[]) => {
    // 選択中の日数を消したら、必ず候補に残る日数を選び直す (design.md D10)。
    const shoppingDefault = next.includes(draft.shoppingDefault)
      ? draft.shoppingDefault
      : constraints.shoppingFallbackDays;
    update({ shoppingPresets: next, shoppingDefault });
  };

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <Section title={t("pages.settings.section.display")}>
        <SelectField
          label={t("pages.settings.field.language")}
          hint={t("pages.settings.hint.language")}
          value={draft.language}
          onChange={(language) => update({ language })}
          options={LANGUAGE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
          errorKey={errorFor("language")}
        />
      </Section>

      <Section title={t("pages.settings.section.meal_plan")}>
        <NumberListField
          label={t("pages.settings.field.leftover_presets")}
          hint={t("pages.settings.hint.leftover_presets")}
          values={draft.leftoverPresets}
          onChange={(leftoverPresets) => update({ leftoverPresets })}
          errorKey={errorFor("leftoverPresets")}
        />
        <IntegerField
          label={t("pages.settings.field.recency_days")}
          hint={t("pages.settings.hint.recency_days")}
          suffix={t("pages.settings.suffix.days")}
          value={draft.recencyDays}
          onChange={(recencyDays) => update({ recencyDays })}
          errorKey={errorFor("recencyDays")}
        />
        <WeekdayField
          label={t("pages.settings.field.weekly_fill_days")}
          hint={t("pages.settings.hint.weekly_fill_days")}
          values={draft.weeklyFillDays}
          onChange={(weeklyFillDays) => update({ weeklyFillDays })}
          errorKey={errorFor("weeklyFillDays")}
        />
        <SelectField
          label={t("pages.settings.field.weekly_fill_meal")}
          value={draft.weeklyFillMeal}
          onChange={(weeklyFillMeal) => update({ weeklyFillMeal })}
          options={WEEKLY_FILL_MEALS.map((m) => ({ value: m, label: t(mealLabelKey(m)) }))}
          errorKey={errorFor("weeklyFillMeal")}
        />
        <IntegerField
          label={t("pages.settings.field.weekly_run_days")}
          hint={t("pages.settings.hint.weekly_run_days")}
          suffix={t("pages.settings.suffix.days")}
          value={draft.weeklyRunDays}
          onChange={(weeklyRunDays) => update({ weeklyRunDays })}
          errorKey={errorFor("weeklyRunDays")}
        />
      </Section>

      <Section title={t("pages.settings.section.fridge")}>
        <IntegerField
          label={t("pages.settings.field.expiry_red")}
          hint={t("pages.settings.hint.expiry_red")}
          suffix={t("pages.settings.suffix.days")}
          value={draft.expiryRed}
          onChange={(expiryRed) => update({ expiryRed })}
          errorKey={errorFor("expiryRed")}
        />
        <IntegerField
          label={t("pages.settings.field.expiry_yellow")}
          hint={t("pages.settings.hint.expiry_yellow")}
          suffix={t("pages.settings.suffix.days")}
          value={draft.expiryYellow}
          onChange={(expiryYellow) => update({ expiryYellow })}
          errorKey={errorFor("expiryYellow")}
        />
      </Section>

      <Section title={t("pages.settings.section.shopping")}>
        <NumberListField
          label={t("pages.settings.field.shopping_presets")}
          hint={t("pages.settings.hint.shopping_presets")}
          values={draft.shoppingPresets}
          lockedValues={constraints.shoppingRequiredDays}
          onChange={onShoppingPresetsChange}
          errorKey={errorFor("shoppingPresets")}
        />
        <SelectField
          label={t("pages.settings.field.shopping_default")}
          value={String(draft.shoppingDefault)}
          onChange={(v) => update({ shoppingDefault: Number(v) })}
          options={draft.shoppingPresets.map((d) => ({
            value: String(d),
            label: t("pages.settings.day_chip", { count: d }),
          }))}
          errorKey={errorFor("shoppingDefault")}
        />
      </Section>

      <Section title={t("pages.settings.section.recipe")}>
        <WordListField
          label={t("pages.settings.field.measuring_units")}
          hint={t("pages.settings.hint.measuring_units")}
          placeholder={t("pages.settings.add_placeholder.unit")}
          values={draft.measuringUnits}
          onChange={(measuringUnits) => update({ measuringUnits })}
          lengthErrorKey="pages.settings.errors.unit_length"
          duplicateErrorKey="pages.settings.errors.unit_duplicate"
          errorKey={errorFor("measuringUnits")}
        />
        <WordListField
          label={t("pages.settings.field.unquantified_words")}
          hint={t("pages.settings.hint.unquantified_words")}
          placeholder={t("pages.settings.add_placeholder.word")}
          values={draft.unquantifiedWords}
          onChange={(unquantifiedWords) => update({ unquantifiedWords })}
          lengthErrorKey="pages.settings.errors.word_length"
          duplicateErrorKey="pages.settings.errors.word_duplicate"
          errorKey={errorFor("unquantifiedWords")}
        />
        <IntegerField
          label={t("pages.settings.field.import_servings")}
          hint={t("pages.settings.hint.import_servings")}
          suffix={t("pages.settings.suffix.servings")}
          value={draft.importServings}
          onChange={(importServings) => update({ importServings })}
          errorKey={errorFor("importServings")}
        />
      </Section>

      <div className="flex items-center justify-between border-t border-neutral-200 pt-4">
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={() => setConfirmOpen(true)}
        >
          {t("pages.settings.reset")}
        </Button>
        <Button type="button" disabled={!dirty || pending} onClick={onSave}>
          {t("pages.settings.save")}
        </Button>
      </div>

      <Dialog
        open={confirmOpen}
        onOpenChange={(next) => {
          if (!next && !reset.isPending) setConfirmOpen(false);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("pages.settings.reset_confirm.title")}</DialogTitle>
            <DialogDescription>{t("pages.settings.reset_confirm.body")}</DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={reset.isPending}
              onClick={() => setConfirmOpen(false)}
            >
              {t("pages.settings.reset_confirm.cancel")}
            </Button>
            <Button type="button" disabled={reset.isPending} onClick={onReset}>
              {t("pages.settings.reset_confirm.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-4 rounded-lg border border-neutral-200 bg-white p-5">
      <h2 className="text-base font-semibold text-neutral-900">{title}</h2>
      {children}
    </section>
  );
}

const WEEKLY_FILL_MEALS = ["breakfast", "lunch", "dinner"] as const;

// 曜日の名前は表示言語の Intl に任せる。2024-01-01 は月曜なので、そこから 1〜7 日目を月〜日に当てる。
function weekdayName(day: number, language: string): string {
  return new Intl.DateTimeFormat(language, { weekday: "short" }).format(new Date(2024, 0, day));
}

function WeekdayField({
  label,
  hint,
  values,
  onChange,
  errorKey,
}: {
  label: string;
  hint: string;
  values: number[];
  onChange: (v: number[]) => void;
  errorKey?: string;
}) {
  const { i18n } = useTranslation();
  return (
    <fieldset className="flex flex-col gap-2" aria-label={label}>
      <legend className="text-sm font-medium text-neutral-800">{label}</legend>
      <div className="flex flex-wrap gap-3">
        {[1, 2, 3, 4, 5, 6, 7].map((day) => (
          <label key={day} className="flex items-center gap-1 text-sm">
            <input
              type="checkbox"
              checked={values.includes(day)}
              onChange={(e) =>
                onChange(e.target.checked ? [...values, day] : values.filter((d) => d !== day))
              }
            />
            {weekdayName(day, i18n.language)}
          </label>
        ))}
      </div>
      <p className="text-xs text-neutral-500">{hint}</p>
      <FieldError messageKey={errorKey} />
    </fieldset>
  );
}

function IntegerField({
  label,
  hint,
  suffix,
  value,
  onChange,
  errorKey,
}: {
  label: string;
  hint: string;
  suffix: string;
  value: string;
  onChange: (v: string) => void;
  errorKey?: string;
}) {
  const id = useId();
  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex items-center gap-2">
        <Input
          id={id}
          type="number"
          step={1}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={errorKey ? true : undefined}
          className="w-24 tabular-nums"
        />
        <span className="text-sm text-neutral-700">{suffix}</span>
      </div>
      <FieldError messageKey={errorKey} />
      <p className="text-xs text-neutral-500">{hint}</p>
    </div>
  );
}

function SelectField({
  label,
  hint,
  value,
  onChange,
  options,
  errorKey,
}: {
  label: string;
  hint?: string;
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
  errorKey?: string;
}) {
  const id = useId();
  return (
    <div className="flex flex-col gap-2">
      <span id={id} className="text-sm font-medium text-neutral-800">
        {label}
      </span>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger aria-labelledby={id} className="w-44">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <FieldError messageKey={errorKey} />
      {hint && <p className="text-xs text-neutral-500">{hint}</p>}
    </div>
  );
}
