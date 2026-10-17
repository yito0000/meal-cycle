import { formatAmount } from "./format-amount";

// 量る単位はレシピの表記に合わせて「大さじ 1」の順にする (add-measuring-units design.md D6)。
// 量る単位は設定で変わるので呼び出し側が渡す。純関数のままにしてクリップボード用の文字列生成からも呼べるようにする
// (add-settings-screen design.md D9)。
export function formatQuantity(
  amount: number,
  unit: string,
  measuringUnits: readonly string[],
): string {
  const u = unit.trim();
  if (amount === 0) return u;
  if (!u) return formatAmount(amount);
  return measuringUnits.includes(u)
    ? `${u} ${formatAmount(amount)}`
    : `${formatAmount(amount)} ${u}`;
}
