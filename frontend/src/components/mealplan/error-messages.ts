// Go 側のエラー sentinel (メッセージ文字列) を i18n key に変換する。
// Wails bindings は Go の error 値を JSON でくるんでフロントに返し、Error.message に
// domain の sentinel が入る。E2E の shim は Error.name に kind を載せている。
// mapping は「message に含まれるサブ文字列」で判定するので、Go 側の文言変更に注意する。

const KEY_UNKNOWN = "pages.meal_plan.errors.unknown";

const RULES: [string, string][] = [
  ["mealplan: invalid date", "pages.meal_plan.errors.invalid_date"],
  ["mealplan: invalid meal", "pages.meal_plan.errors.invalid_meal"],
  ["mealplan: invalid position", "pages.meal_plan.errors.invalid_position"],
  // MergeBatch のみ「合流先バッチが同 meal に無い」の文脈で ErrEntryNotFound を返すため、
  // 判別できないメッセージ側で先に load される順序に頼らない (先勝ちで entry_not_found)。
  ["mealplan: entry not found", "pages.meal_plan.errors.entry_not_found"],
  ["mealplan: reorder mismatch", "pages.meal_plan.errors.reorder_mismatch"],
  ["mealplan: invalid copy days", "pages.meal_plan.errors.invalid_copy_days"],
  ["mealplan: invalid copy mode", "pages.meal_plan.errors.unknown"],
  ["mealplan: unknown recipe", "pages.meal_plan.errors.unknown_recipe"],
  ["mealplan: batch target not found", "pages.meal_plan.errors.batch_target_not_found"],
  ["mealplan: batch merge denied", "pages.meal_plan.errors.batch_merge_denied"],
  ["mealplan: invalid period weeks", "pages.meal_plan.errors.invalid_period_weeks"],
];

export function errorI18nKey(err: unknown): string {
  if (!(err instanceof Error)) return KEY_UNKNOWN;
  const msg = err.message ?? "";
  for (const [needle, key] of RULES) {
    if (msg.includes(needle)) return key;
  }
  return KEY_UNKNOWN;
}
