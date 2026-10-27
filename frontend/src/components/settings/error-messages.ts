export type SettingsField =
  | "language"
  | "expiryRed"
  | "expiryYellow"
  | "leftoverPresets"
  | "shoppingPresets"
  | "shoppingDefault"
  | "measuringUnits"
  | "unquantifiedWords"
  | "importServings"
  | "recencyDays"
  | "weeklyFillDays"
  | "weeklyFillMeal"
  | "weeklyRunDays";

const P = "pages.settings.errors.";

// Go 側の sentinel (internal/domain/settings/errors.go) のメッセージを i18n key と欄に対応付ける。
// 部分文字列で判定するので、Go 側の文言を変えたらここも直す。
const RULES: [string, string, SettingsField][] = [
  ["settings: invalid language", `${P}invalid_language`, "language"],
  ["settings: red days out of range", `${P}red_out_of_range`, "expiryRed"],
  ["settings: yellow days out of range", `${P}yellow_out_of_range`, "expiryYellow"],
  ["settings: red days not below yellow days", `${P}red_not_below_yellow`, "expiryRed"],
  ["settings: leftover presets count", `${P}leftover_count`, "leftoverPresets"],
  ["settings: leftover preset out of range", `${P}leftover_out_of_range`, "leftoverPresets"],
  ["settings: leftover preset duplicate", `${P}leftover_duplicate`, "leftoverPresets"],
  ["settings: shopping presets count", `${P}shopping_count`, "shoppingPresets"],
  ["settings: shopping preset out of range", `${P}shopping_out_of_range`, "shoppingPresets"],
  ["settings: shopping preset duplicate", `${P}shopping_duplicate`, "shoppingPresets"],
  [
    "settings: shopping preset required missing",
    `${P}shopping_required_missing`,
    "shoppingPresets",
  ],
  [
    "settings: shopping default not in presets",
    `${P}shopping_default_not_in_presets`,
    "shoppingDefault",
  ],
  ["settings: measuring units count", `${P}units_count`, "measuringUnits"],
  ["settings: measuring unit length", `${P}unit_length`, "measuringUnits"],
  ["settings: measuring unit duplicate", `${P}unit_duplicate`, "measuringUnits"],
  ["settings: unquantified words count", `${P}words_count`, "unquantifiedWords"],
  ["settings: unquantified word length", `${P}word_length`, "unquantifiedWords"],
  ["settings: unquantified word duplicate", `${P}word_duplicate`, "unquantifiedWords"],
  ["settings: import servings out of range", `${P}servings_out_of_range`, "importServings"],
  ["settings: recency days out of range", `${P}recency_out_of_range`, "recencyDays"],
  ["settings: weekly fill days empty", `${P}weekly_days_empty`, "weeklyFillDays"],
  ["settings: weekly fill day out of range", `${P}weekly_day_out_of_range`, "weeklyFillDays"],
  ["settings: weekly fill day duplicate", `${P}weekly_day_out_of_range`, "weeklyFillDays"],
  ["settings: weekly fill invalid meal", `${P}weekly_meal_invalid`, "weeklyFillMeal"],
  ["settings: weekly run days out of range", `${P}weekly_run_out_of_range`, "weeklyRunDays"],
];

export type SettingsError = { key: string; field: SettingsField | null };

export function settingsErrorOf(err: unknown): SettingsError {
  const msg = err instanceof Error ? (err.message ?? "") : "";
  for (const [needle, key, field] of RULES) {
    if (msg.includes(needle)) return { key, field };
  }
  return { key: `${P}unknown`, field: null };
}
