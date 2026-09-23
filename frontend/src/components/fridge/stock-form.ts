import type * as React from "react";
import {
  ExpiryStatus,
  type StockItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";

// internal/service/fridge.go の検証上限と揃える。Go 側が最終防衛線 (design.md D11)。
const NAME_MAX = 100;
const AMOUNT_MAX = 99999;
const UNIT_MAX = 20;

export type StockField = "name" | "amount" | "unit" | "expiresAt";

// 量は入力途中の "" や "1." を保持するため文字列で持つ。
export type StockFormValues = {
  name: string;
  amount: string;
  unit: string;
  expiresAt: string;
};

export type StockFormErrors = Partial<Record<StockField, string>>;

export const emptyStockForm: StockFormValues = { name: "", amount: "", unit: "", expiresAt: "" };

export function fromStockItem(item: StockItem): StockFormValues {
  return {
    name: item.name,
    amount: String(item.amount),
    unit: item.unit,
    expiresAt: item.expiresAt,
  };
}

function runeLength(s: string): number {
  return [...s].length;
}

function isValidDate(s: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return false;
  const [y, m, d] = s.split("-").map(Number);
  const date = new Date(Date.UTC(y, m - 1, d));
  return date.getUTCFullYear() === y && date.getUTCMonth() === m - 1 && date.getUTCDate() === d;
}

// 戻り値は i18n key。空オブジェクトなら送信してよい。
export function validateStockForm(values: StockFormValues): StockFormErrors {
  const errors: StockFormErrors = {};
  const name = values.name.trim();
  if (name === "") errors.name = "pages.fridge.errors.name_empty";
  else if (runeLength(name) > NAME_MAX) errors.name = "pages.fridge.errors.name_too_long";

  const amount = values.amount.trim() === "" ? Number.NaN : Number(values.amount);
  if (!Number.isFinite(amount) || amount <= 0 || amount > AMOUNT_MAX) {
    errors.amount = "pages.fridge.errors.amount_out_of_range";
  }

  if (runeLength(values.unit.trim()) > UNIT_MAX) errors.unit = "pages.fridge.errors.unit_too_long";

  const expiresAt = values.expiresAt.trim();
  if (expiresAt !== "" && !isValidDate(expiresAt)) {
    errors.expiresAt = "pages.fridge.errors.invalid_expiry";
  }
  return errors;
}

// ID / 日時 / 算出値は Go 側で決まるので空値で送る。
export function toStockInput(values: StockFormValues): StockItem {
  return {
    id: "",
    name: values.name.trim(),
    amount: Number(values.amount),
    unit: values.unit.trim(),
    expiresAt: values.expiresAt.trim(),
    addedAt: "",
    updatedAt: "",
    expiryStatus: ExpiryStatus.$zero,
    daysLeft: null,
  };
}

// IME の変換確定の Enter を送信や候補確定と取り違えないための判定。WebKit 系は
// compositionend 直後に isComposing=false の keydown が来ることがあり、keyCode 229 を併用する (design.md D6)。
export function isImeComposing(e: React.KeyboardEvent): boolean {
  return e.nativeEvent.isComposing || e.keyCode === 229;
}
