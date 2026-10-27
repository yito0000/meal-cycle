import type { StockField } from "./stock-form";

// Go 側の sentinel (internal/domain/fridge/errors.go) のメッセージを i18n key と入力欄に対応付ける。
// 部分文字列で判定するので、Go 側の文言を変えたらここも直す。
const RULES: [string, string, StockField | null][] = [
  ["fridge: name empty", "pages.fridge.errors.name_empty", "name"],
  ["fridge: name too long", "pages.fridge.errors.name_too_long", "name"],
  ["fridge: amount out of range", "pages.fridge.errors.amount_out_of_range", "amount"],
  ["fridge: unit too long", "pages.fridge.errors.unit_too_long", "unit"],
  ["fridge: invalid expiry date", "pages.fridge.errors.invalid_expiry", "expiresAt"],
  ["fridge: stock item not found", "pages.fridge.errors.not_found", null],
  [
    "fridge: consume amount out of range",
    "pages.meal_plan.consume.errors.amount_out_of_range",
    null,
  ],
];

const KEY_UNKNOWN = "pages.fridge.errors.unknown";

export function stockErrorOf(err: unknown): { key: string; field: StockField | null } {
  const msg = err instanceof Error ? (err.message ?? "") : "";
  for (const [needle, key, field] of RULES) {
    if (msg.includes(needle)) return { key, field };
  }
  return { key: KEY_UNKNOWN, field: null };
}

// 常備品のコードは "fridge: staple ..." で、在庫の RULES より先に判定する (add-fridge-staples design.md D4)。
const STAPLE_RULES: [string, string][] = [
  ["fridge: staple name empty", "pages.fridge.staples.errors.name_empty"],
  ["fridge: staple name too long", "pages.fridge.staples.errors.name_too_long"],
  ["fridge: staple duplicate", "pages.fridge.staples.errors.duplicate"],
  ["fridge: staple invalid state", "pages.fridge.staples.errors.invalid_state"],
  ["fridge: staple not found", "pages.fridge.staples.errors.not_found"],
];

export function stapleErrorOf(err: unknown): string {
  const msg = err instanceof Error ? (err.message ?? "") : "";
  for (const [needle, key] of STAPLE_RULES) {
    if (msg.includes(needle)) return key;
  }
  return "pages.fridge.staples.errors.unknown";
}
