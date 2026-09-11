import { BATCH_COLORS } from "./mealplan-config";

// colorForBatch は batch_id を決定的に BATCH_COLORS の 1 色に写す。
// memberCount <= 1 は「単独バッチ = 色付けしない」で null を返す (spec の視覚識別要件)。
// 決定性を保つのは「週をまたいで同じバッチ → 同じ色」を維持するため。
export function colorForBatch(batchID: string, memberCount: number): string | null {
  if (!batchID || memberCount <= 1) return null;
  let hash = 0;
  for (let i = 0; i < batchID.length; i++) {
    hash = (hash * 31 + batchID.charCodeAt(i)) >>> 0;
  }
  return BATCH_COLORS[hash % BATCH_COLORS.length];
}
