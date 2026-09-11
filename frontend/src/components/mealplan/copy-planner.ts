import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";

// previewCopy はコピー先の日付を列挙し、そのうち既存割当がある日付 (衝突) を返す。
// 「連続 N 日」の解釈は起点日を含む合計 N 日 (design.md D5)。起点日は反映対象から除外する
// (作り置きは翌日以降に伸ばす目的なので)。
export function previewCopy(
  sourceDate: string,
  slot: string,
  days: number,
  allEntries: Entry[],
): { targets: string[]; conflicts: string[] } {
  const targets = enumerateTargets(sourceDate, days);
  const occupied = new Set<string>();
  for (const e of allEntries) {
    if (e.slot === slot) occupied.add(e.date);
  }
  const conflicts = targets.filter((d) => occupied.has(d));
  return { targets, conflicts };
}

function enumerateTargets(sourceDate: string, days: number): string[] {
  if (days <= 1) return [];
  const parts = sourceDate.split("-").map(Number);
  const t = new Date(parts[0], parts[1] - 1, parts[2]);
  const out: string[] = [];
  for (let i = 1; i < days; i++) {
    const d = new Date(t.getFullYear(), t.getMonth(), t.getDate() + i);
    out.push(formatLocalDate(d));
  }
  return out;
}

function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}
