// 量の表示。整数なら小数点を出さない。S05 と S07 で同じ見え方にするため共有する。
export function formatAmount(amount: number): string {
  return Number.isInteger(amount) ? String(amount) : String(Number(amount.toFixed(2)));
}
