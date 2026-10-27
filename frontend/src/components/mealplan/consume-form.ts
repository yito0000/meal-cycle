import { formatAmount } from "@/lib/format-amount";
import type {
  Consumption,
  ConsumptionCandidate,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";

export type ConsumeRow = {
  stockId: string;
  expiresAt: string;
  current: number;
  amount: string;
};

// .ts / .js 生成で nullable の付き方が違うので、読むフィールドだけの緩い形で受ける。
type PlanLike = {
  lines:
    | {
        candidates:
          | Pick<ConsumptionCandidate, "stockId" | "expiresAt" | "currentAmount" | "amount">[]
          | null;
      }[]
    | null;
};

const KEY_OUT_OF_RANGE = "pages.meal_plan.consume.errors.amount_out_of_range";

// 行は材料ごとの候補を順につなげたもの。ダイアログは同じ順で材料の見出しの下に並べる。
export function toConsumeRows(plan: PlanLike): ConsumeRow[] {
  return (plan.lines ?? []).flatMap((l) =>
    (l.candidates ?? []).map((c) => ({
      stockId: c.stockId,
      expiresAt: c.expiresAt,
      current: c.currentAmount,
      amount: formatAmount(c.amount),
    })),
  );
}

function parseAmount(s: string): number | null {
  const t = s.trim();
  if (t === "") return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}

// Go 側と同じく小数第 2 位で比べ、0.1+0.2 の誤差で「今の量を超える」に倒れないようにする。
export function validateConsumeRows(rows: ConsumeRow[]): {
  errors: (string | undefined)[];
  hasError: boolean;
} {
  const errors = rows.map((r) => {
    const n = parseAmount(r.amount);
    if (n === null || n < 0 || Math.round(n * 100) > Math.round(r.current * 100)) {
      return KEY_OUT_OF_RANGE;
    }
    return undefined;
  });
  return { errors, hasError: errors.some((e) => e !== undefined) };
}

export function toConsumptions(rows: ConsumeRow[]): Consumption[] {
  return rows
    .map((r) => ({ stockId: r.stockId, amount: Number(r.amount.trim()) }))
    .filter((c) => c.amount > 0);
}
