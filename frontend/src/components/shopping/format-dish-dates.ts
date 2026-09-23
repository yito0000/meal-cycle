import type { TFunction } from "i18next";

function parts(date: string): { y: number; m: number; d: number } {
  const [y, m, d] = date.split("-").map(Number);
  return { y, m, d };
}

function isNextDay(prev: string, next: string): boolean {
  const a = parts(prev);
  const b = parts(next);
  const t = new Date(a.y, a.m - 1, a.d + 1);
  return t.getFullYear() === b.y && t.getMonth() + 1 === b.m && t.getDate() === b.d;
}

// 連続なら「9/24〜26」(月をまたぐときは終端にも月を付けて「9/30〜10/2」)、飛び飛びなら「9/24, 9/26」。
export function formatDishDates(dates: string[], t: TFunction): string {
  if (dates.length === 0) return "";
  const first = parts(dates[0]);
  const label = `${first.m}/${first.d}`;
  if (dates.length === 1) return label;
  const contiguous = dates.every((d, i) => i === 0 || isNextDay(dates[i - 1], d));
  if (!contiguous) {
    return dates
      .map((d) => {
        const p = parts(d);
        return `${p.m}/${p.d}`;
      })
      .join(", ");
  }
  const last = parts(dates[dates.length - 1]);
  const end = last.m === first.m ? String(last.d) : `${last.m}/${last.d}`;
  return t("pages.shopping.dishes.date_range", { start: label, end });
}
