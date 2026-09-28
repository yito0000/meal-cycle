import { formatAmount } from "./format-amount";

// internal/config/defaults.go (MeasuringUnits) のフロント側 mirror。
// S09 で ConfigService が Defaults() を返すまで手動で同期する。source of truth は defaults_test.go。
export const MEASURING_UNITS: readonly string[] = ["大さじ", "小さじ", "カップ"];

// 量る単位はレシピの表記に合わせて「大さじ 1」の順にする (add-measuring-units design.md D6)。
export function formatQuantity(amount: number, unit: string): string {
  const u = unit.trim();
  if (amount === 0) return u;
  if (!u) return formatAmount(amount);
  return MEASURING_UNITS.includes(u)
    ? `${u} ${formatAmount(amount)}`
    : `${formatAmount(amount)} ${u}`;
}
